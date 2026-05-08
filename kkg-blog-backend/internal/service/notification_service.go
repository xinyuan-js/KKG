package service

import (
	"errors"

	"awesomeProject/internal/model"
	"awesomeProject/internal/repository"
)

type NotificationList struct {
	UnreadCount int64                `json:"unread_count"`
	Items       []model.Notification `json:"items"`
}

type NotificationService struct {
	notifications *repository.NotificationRepository
}

func NewNotificationService(notifications *repository.NotificationRepository) *NotificationService {
	return &NotificationService{notifications: notifications}
}

func (s *NotificationService) ListMine(userID uint64, limit int) (*NotificationList, error) {
	if userID == 0 {
		return nil, errors.New("invalid user")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items, err := s.notifications.ListByReceiver(userID, limit)
	if err != nil {
		return nil, err
	}
	unread, err := s.notifications.CountUnreadByReceiver(userID)
	if err != nil {
		return nil, err
	}
	return &NotificationList{UnreadCount: unread, Items: items}, nil
}

func (s *NotificationService) MarkRead(userID uint64, notificationID uint64) error {
	if userID == 0 || notificationID == 0 {
		return errors.New("invalid request")
	}
	n, err := s.notifications.GetByID(notificationID)
	if err != nil {
		return err
	}
	if n == nil {
		return errors.New("notification not found")
	}
	if n.ReceiverID != userID {
		return errors.New("forbidden")
	}
	if n.IsRead {
		return nil
	}
	return s.notifications.MarkRead(notificationID)
}
