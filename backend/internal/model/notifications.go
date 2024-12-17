package model

import "encoding/json"

type NotificationType string

const (
	InvalidateAllNotificationType NotificationType = "InvalidateAll"
	AddFriendNotificationType     NotificationType = "AddFriend"
	DeleteFriendNotificationType  NotificationType = "DeleteFriend"
	CreatePostNotificationType    NotificationType = "CreatePost"
	UpdatePostNotificationType    NotificationType = "UpdatePost"
	DeletePostNotificationType    NotificationType = "DeletePost"
)

type Notification struct {
	Type    NotificationType `json:"notification_type"`
	Payload any              `json:"payload"`
}

type NotificationJson struct {
	Type    NotificationType `json:"notification_type"`
	Payload json.RawMessage  `json:"payload"`
}

type InvalidateAllNotification struct {
}

type AddFriendNotification struct {
	UserId   UserId `json:"user_id"`
	FriendId UserId `json:"friend_id"`
}

type DeleteFriendNotification struct {
	UserId   UserId `json:"user_id"`
	FriendId UserId `json:"friend_id"`
}

type CreatePostNotification struct {
	UserId UserId       `json:"user_id"`
	Post   PostExtended `json:"post"`
}

type UpdatePostNotification struct {
	UserId UserId       `json:"user_id"`
	Post   PostExtended `json:"post"`
}

type DeletePostNotification struct {
	UserId UserId `json:"user_id"`
	PostId PostId `json:"post_id"`
}
