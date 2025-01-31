package model

import "time"

type DialogId = string

type DialogMessageExtended struct {
	DialogMessage
	Id       string
	DialogId DialogId
	SendTime time.Time
}
