package model

import "time"

type UserExtended struct {
	User
	PasswordHash []byte `db:"password_hash"`
}

type PostExtended struct {
	Post
	CreateTime time.Time
	UpdateTime time.Time
}

type DialogId = string

type DialogMessageExtended struct {
	DialogMessage
	Id       string
	DialogId DialogId
	SendTime time.Time
}
