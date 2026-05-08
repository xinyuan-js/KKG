package mq

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type JudgeMessage struct {
	SubmitID int64 `json:"submitId"`
}

type JudgeQueue struct {
	conn      *amqp.Connection
	ch        *amqp.Channel
	queueName string
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
	if _, err = ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	return &JudgeQueue{conn: conn, ch: ch, queueName: queueName}, nil
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

func (q *JudgeQueue) Publish(submitID int64) error {
	if q == nil {
		return fmt.Errorf("queue is nil")
	}
	body, _ := json.Marshal(JudgeMessage{SubmitID: submitID})
	return q.ch.PublishWithContext(context.Background(), "", q.queueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
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
					_ = d.Nack(false, true)
					continue
				}
				_ = d.Ack(false)
			}
		}
	}()
	return nil
}

