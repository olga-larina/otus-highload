package model

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrNotValidPassword    = errors.New("not valid password")
	ErrNotValidCredentials = errors.New("not valid credentials")
	ErrClaimsInvalid       = errors.New("provided claims do not match expected scopes")
)
