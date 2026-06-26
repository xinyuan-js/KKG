package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	defaultMaxRetries = 3
	defaultRetryDelay = 5 * time.Second
)

type JudgeMessage struct {
	SubmitID int64 `json:"submitId"`
}

type JudgeQueue struct {
	conn      *amqp.Connection
	ch        *amqp.Channel
	queueName string
	retryName string
	dlqName   string
	confirms  <-chan amqp.Confirmation
	onDead    func(submitID int64, reason string, retryCount int32)
	mu        sync.Mutex
}

func NewJudgeQueue(url, queueName string) (*JudgeQueue, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	retryName := queueName + ".retry"
	dlqName := queueName + ".dlq"
	if _, err = ch.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	if _, err = ch.QueueDeclare(retryName, true, false, false, false, amqp.Table{
		"x-message-ttl":             int32(defaultRetryDelay / time.Millisecond),
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": queueName,
	}); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	if _, err = ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	if err = ch.Confirm(false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	return &JudgeQueue{
		conn:      conn,
		ch:        ch,
		queueName: queueName,
		retryName: retryName,
		dlqName:   dlqName,
		confirms:  ch.NotifyPublish(make(chan amqp.Confirmation, 1)),
	}, nil
}

func (q *JudgeQueue) Close() error {
	if q == nil {
		return nil
	}
	if q.ch != nil {
		_ = q.ch.Close()
	}
	if q.conn != nil {
		return q.conn.Close()
	}
	return nil
}

func (q *JudgeQueue) SetDeadLetterHandler(fn func(submitID int64, reason string, retryCount int32)) {
	if q == nil {
		return
	}
	q.onDead = fn
}

func (q *JudgeQueue) Publish(submitID int64) error {
	if q == nil {
		return fmt.Errorf("queue is nil")
	}
	body, _ := json.Marshal(JudgeMessage{SubmitID: submitID})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return q.publish(ctx, q.queueName, body, nil)
}

func (q *JudgeQueue) publish(ctx context.Context, routingKey string, body []byte, headers amqp.Table) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if err := q.ch.PublishWithContext(ctx, "", routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Headers:      headers,
		Body:         body,
	}); err != nil {
		return err
	}
	select {
	case confirmation, ok := <-q.confirms:
		if !ok {
			return fmt.Errorf("publisher confirm channel closed")
		}
		if !confirmation.Ack {
			return fmt.Errorf("message publish rejected")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *JudgeQueue) Consume(ctx context.Context, handler func(submitID int64) error) error {
	if q == nil {
		return fmt.Errorf("queue is nil")
	}
	if err := q.ch.Qos(1, 0, false); err != nil {
		return err
	}
	msgs, err := q.ch.Consume(q.queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-msgs:
				if !ok {
					return
				}
				var m JudgeMessage
				if err = json.Unmarshal(d.Body, &m); err != nil || m.SubmitID <= 0 {
					_ = d.Ack(false)
					continue
				}
				if hErr := handler(m.SubmitID); hErr != nil {
					if err = q.retryOrDeadLetter(ctx, d, m, hErr); err != nil {
						_ = d.Nack(false, true)
						continue
					}
					_ = d.Ack(false)
					continue
				}
				_ = d.Ack(false)
			}
		}
	}()
	return nil
}

func (q *JudgeQueue) retryOrDeadLetter(ctx context.Context, d amqp.Delivery, m JudgeMessage, reason error) error {
	retryCount := deliveryRetryCount(d)
	body, _ := json.Marshal(m)
	headers := amqp.Table{
		"retry-count": retryCount + 1,
		"last-error":  reason.Error(),
	}
	target := q.retryName
	isDeadLetter := retryCount >= defaultMaxRetries
	if isDeadLetter {
		target = q.dlqName
		headers["dead-letter-reason"] = reason.Error()
	}
	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := q.publish(pubCtx, target, body, headers); err != nil {
		return err
	}
	if isDeadLetter && q.onDead != nil {
		q.onDead(m.SubmitID, reason.Error(), retryCount+1)
	}
	return nil
}

func deliveryRetryCount(d amqp.Delivery) int32 {
	if d.Headers == nil {
		return 0
	}
	switch v := d.Headers["retry-count"].(type) {
	case int32:
		return v
	case int64:
		return int32(v)
	case int:
		return int32(v)
	case uint8:
		return int32(v)
	case string:
		var n int32
		_, _ = fmt.Sscanf(v, "%d", &n)
		return n
	default:
		return 0
	}
}
