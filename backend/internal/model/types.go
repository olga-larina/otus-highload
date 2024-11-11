package model

type UserExtended struct {
	User
	PasswordHash []byte `db:"password_hash"`
}
