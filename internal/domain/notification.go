package domain

import "slices"

type NotificationType string

const (
	NotificationChannelInApp     NotificationChannel = "in_app"
	NotificationBookingCreated   NotificationType    = "booking_created"
	NotificationBookingConfirmed NotificationType    = "booking_confirmed"
	NotificationBookingCancelled NotificationType    = "booking_cancelled"
	NotificationBookingReminder  NotificationType    = "booking_reminder"
)

var notificationTypes = []NotificationType{
	NotificationBookingCreated, NotificationBookingConfirmed,
	NotificationBookingCancelled, NotificationBookingReminder,
}

func ParseNotificationType(s string) (NotificationType, error) {
	return parseEnum("notification type", s, notificationTypes)
}

func (t NotificationType) Valid() bool { return slices.Contains(notificationTypes, t) }

type NotificationChannel string

const (
	NotificationChannelPush     NotificationChannel = "push"
	NotificationChannelSMS      NotificationChannel = "sms"
	NotificationChannelWhatsApp NotificationChannel = "whatsapp"
	NotificationChannelTelegram NotificationChannel = "telegram"
	NotificationChannelEmail    NotificationChannel = "email"
)

var notificationChannels = []NotificationChannel{
	NotificationChannelInApp,
	NotificationChannelPush, NotificationChannelSMS, NotificationChannelWhatsApp,
	NotificationChannelTelegram, NotificationChannelEmail,
}

func ParseNotificationChannel(s string) (NotificationChannel, error) {
	return parseEnum("notification channel", s, notificationChannels)
}

func (c NotificationChannel) Valid() bool { return slices.Contains(notificationChannels, c) }

type NotificationStatus string

const (
	// NotificationPending is also hard-coded in the notifications_dispatch_idx partial index.
	NotificationPending    NotificationStatus = "pending"
	NotificationProcessing NotificationStatus = "processing"
	NotificationSent       NotificationStatus = "sent"
	NotificationFailed     NotificationStatus = "failed"
	NotificationCancelled  NotificationStatus = "cancelled"
)

var notificationStatuses = []NotificationStatus{
	NotificationPending, NotificationProcessing, NotificationSent,
	NotificationFailed, NotificationCancelled,
}

func ParseNotificationStatus(s string) (NotificationStatus, error) {
	return parseEnum("notification status", s, notificationStatuses)
}

func (s NotificationStatus) Valid() bool { return slices.Contains(notificationStatuses, s) }
