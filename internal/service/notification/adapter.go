package notification

import (
	"computer-management-api/internal/notification"
	"computer-management-api/internal/service"
	"context"
	"fmt"
)

// ServiceAdapter adapts the notification client to the service layer interface
type ServiceAdapter struct {
	client notification.Notifier
}

// NewServiceAdapter creates a new notification service adapter
func NewServiceAdapter(client notification.Notifier) *ServiceAdapter {
	return &ServiceAdapter{
		client: client,
	}
}

// SendComputerNotification sends a computer-related notification
func (a *ServiceAdapter) SendComputerNotification(ctx context.Context, computerNotification service.ComputerNotification) error {
	// Convert service notification to client notification
	clientNotification := notification.Notification{
		Level:                mapNotificationLevel(computerNotification.Type),
		EmployeeAbbreviation: computerNotification.EmployeeAbbreviation,
		Message:              computerNotification.Message,
		Metadata:             computerNotification.Metadata,
	}

	// Add computer-specific metadata
	if computerNotification.ComputerName != "" {
		if clientNotification.Metadata == nil {
			clientNotification.Metadata = make(map[string]string)
		}
		clientNotification.Metadata["computer_name"] = computerNotification.ComputerName
	}

	if computerNotification.ComputerCount > 0 {
		if clientNotification.Metadata == nil {
			clientNotification.Metadata = make(map[string]string)
		}
		clientNotification.Metadata["computer_count"] = fmt.Sprintf("%d", computerNotification.ComputerCount)
	}

	clientNotification.Metadata["notification_type"] = string(computerNotification.Type)

	return a.client.SendNotificationWithContext(ctx, clientNotification)
}

// mapNotificationLevel maps service notification types to client notification levels
func mapNotificationLevel(notificationType service.NotificationType) notification.NotificationLevel {
	switch notificationType {
	case service.NotificationTypeThresholdExceeded:
		return notification.LevelWarning
	case service.NotificationTypeComputerCreated:
		return notification.LevelInfo
	case service.NotificationTypeComputerUpdated:
		return notification.LevelInfo
	case service.NotificationTypeComputerDeleted:
		return notification.LevelWarning
	default:
		return notification.LevelInfo
	}
}
