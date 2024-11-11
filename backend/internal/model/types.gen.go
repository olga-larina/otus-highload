// Package model provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.1 DO NOT EDIT.
package model

const (
	BearerAuthScopes = "bearerAuth.Scopes"
)

// Defines values for Gender.
const (
	Female Gender = "F"
	Male   Gender = "M"
)

// BirthDate Дата рождения
type BirthDate = Date

// DialogMessage defines model for DialogMessage.
type DialogMessage struct {
	// From Идентификатор пользователя
	From UserId `json:"from"`

	// Text Текст сообщения
	Text DialogMessageText `json:"text"`

	// To Идентификатор пользователя
	To UserId `json:"to"`
}

// DialogMessageText Текст сообщения
type DialogMessageText = string

// Gender Пол
type Gender string

// Post Пост пользователя
type Post struct {
	// AuthorUserId Идентификатор пользователя
	AuthorUserId *UserId `json:"author_user_id,omitempty"`

	// Id Идентификатор поста
	Id *PostId `json:"id,omitempty"`

	// Text Текст поста
	Text *PostText `json:"text,omitempty"`
}

// PostId Идентификатор поста
type PostId = string

// PostText Текст поста
type PostText = string

// User defines model for User.
type User struct {
	// Biography Интересы
	Biography *string `db:"biography" json:"biography,omitempty"`

	// Birthdate Дата рождения
	Birthdate *BirthDate `json:"birthdate,omitempty"`

	// City Город
	City *string `db:"city" json:"city,omitempty"`

	// FirstName Имя
	FirstName *string `db:"first_name" json:"first_name,omitempty"`

	// Gender Пол
	Gender *Gender `json:"gender,omitempty"`

	// Id Идентификатор пользователя
	Id *UserId `json:"id,omitempty"`

	// SecondName Фамилия
	SecondName *string `db:"second_name" json:"second_name,omitempty"`
}

// UserId Идентификатор пользователя
type UserId = string

// N5xx defines model for 5xx.
type N5xx struct {
	// Code Код ошибки. Предназначен для классификации проблем и более быстрого решения проблем.
	Code *int `json:"code,omitempty"`

	// Message Описание ошибки
	Message string `json:"message"`

	// RequestId Идентификатор запроса. Предназначен для более быстрого поиска проблем.
	RequestId *string `json:"request_id,omitempty"`
}

// PostDialogUserIdSendJSONBody defines parameters for PostDialogUserIdSend.
type PostDialogUserIdSendJSONBody struct {
	// Text Текст сообщения
	Text DialogMessageText `json:"text"`
}

// PostLoginJSONBody defines parameters for PostLogin.
type PostLoginJSONBody struct {
	// Id Идентификатор пользователя
	Id       *UserId `json:"id,omitempty"`
	Password *string `json:"password,omitempty"`
}

// PostPostCreateJSONBody defines parameters for PostPostCreate.
type PostPostCreateJSONBody struct {
	// Text Текст поста
	Text PostText `json:"text"`
}

// GetPostFeedParams defines parameters for GetPostFeed.
type GetPostFeedParams struct {
	Offset *float32 `form:"offset,omitempty" json:"offset,omitempty"`
	Limit  *float32 `form:"limit,omitempty" json:"limit,omitempty"`
}

// PutPostUpdateJSONBody defines parameters for PutPostUpdate.
type PutPostUpdateJSONBody struct {
	// Id Идентификатор поста
	Id PostId `json:"id"`

	// Text Текст поста
	Text PostText `json:"text"`
}

// PostUserRegisterJSONBody defines parameters for PostUserRegister.
type PostUserRegisterJSONBody struct {
	Biography *string `json:"biography,omitempty"`

	// Birthdate Дата рождения
	Birthdate *BirthDate `json:"birthdate,omitempty"`
	City      *string    `json:"city,omitempty"`
	FirstName *string    `json:"first_name,omitempty"`

	// Gender Пол
	Gender     *Gender `json:"gender,omitempty"`
	Password   *string `json:"password,omitempty"`
	SecondName *string `json:"second_name,omitempty"`
}

// GetUserSearchParams defines parameters for GetUserSearch.
type GetUserSearchParams struct {
	// FirstName Условие поиска по имени
	FirstName string `form:"first_name" json:"first_name"`

	// LastName Условие поиска по фамилии
	LastName string `form:"last_name" json:"last_name"`
}

// PostDialogUserIdSendJSONRequestBody defines body for PostDialogUserIdSend for application/json ContentType.
type PostDialogUserIdSendJSONRequestBody PostDialogUserIdSendJSONBody

// PostLoginJSONRequestBody defines body for PostLogin for application/json ContentType.
type PostLoginJSONRequestBody PostLoginJSONBody

// PostPostCreateJSONRequestBody defines body for PostPostCreate for application/json ContentType.
type PostPostCreateJSONRequestBody PostPostCreateJSONBody

// PutPostUpdateJSONRequestBody defines body for PutPostUpdate for application/json ContentType.
type PutPostUpdateJSONRequestBody PutPostUpdateJSONBody

// PostUserRegisterJSONRequestBody defines body for PostUserRegister for application/json ContentType.
type PostUserRegisterJSONRequestBody PostUserRegisterJSONBody
