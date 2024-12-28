package model

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrNotValidPassword    = errors.New("not valid password")
	ErrNotValidCredentials = errors.New("not valid credentials")
	ErrNotAuthorized       = errors.New("not authorized")
	ErrClaimsInvalid       = errors.New("provided claims do not match expected scopes")
	ErrPostNotFound        = errors.New("post not found")
	ErrPostFeedLenNotValid = errors.New("post feed length not valid")
	ErrNotValidCache       = errors.New("cache not valid")
	ErrNotValidDialog      = errors.New("dialog not valid")
)
