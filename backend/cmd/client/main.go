package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	clienthttp "github.com/olga-larina/otus-highload/backend/internal/client/http"
	"github.com/olga-larina/otus-highload/backend/internal/logger"
	"github.com/olga-larina/otus-highload/backend/internal/model"
)

func main() {
	var err error

	ctx := context.Background()

	err = logger.New("DEBUG")
	if err != nil {
		log.Fatalf("failed building logger %v", err)
		return
	}

	hc := http.Client{}

	c, err := clienthttp.NewClientWithResponses("http://localhost:8080", clienthttp.WithHTTPClient(&hc))
	if err != nil {
		logger.Error(ctx, err, "failed creating http client")
		return
	}

	var userId, token string
	password := "Password@12345"

	// --------- Проверка корректных запросов

	// регистрация пользователя
	userId, err = userRegister(ctx, c, password)
	if err != nil {
		return
	}

	// получение пользователя по ID
	err = userGet(ctx, c, userId)
	if err != nil {
		return
	}

	// получение токена
	token, err = login(ctx, c, userId, password)
	if err != nil {
		return
	}

	// авторизация
	addTokenFunc := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}

	// получение информации о себе (через метод, требующий токена)
	err = getMe(ctx, c, addTokenFunc)
	if err != nil {
		return
	}

	// --------- Проверка некорректных запросов

	// логин с некорректным паролем
	login(ctx, c, userId, "ABC")

	// несуществующий пользователь
	userGet(ctx, c, "ABC")

	// получение информации о себе без токена
	getMe(ctx, c, func(ctx context.Context, req *http.Request) error { return nil })
}

func userRegister(ctx context.Context, c *clienthttp.ClientWithResponses, password string) (string, error) {
	userRegisterResponse, err := c.PostUserRegisterWithResponse(ctx, model.PostUserRegisterJSONRequestBody{
		FirstName:  ptrString("Имя"),
		SecondName: ptrString("Фамилия"),
		Gender:     ptrGender(model.Female),
		Birthdate:  model.NewDate(1990, 1, 2),
		Biography:  ptrString("Хобби, интересы и т.п."),
		City:       ptrString("Москва"),
		Password:   ptrString(password),
	})
	if err != nil {
		logger.Error(ctx, err, "failed registering user")
		return "", err
	}
	if userRegisterResponse.StatusCode() != 200 {
		logger.Warn(ctx, "failed registering user", "response", string(userRegisterResponse.Body), "status", userRegisterResponse.StatusCode())
		return "", errors.New("failed registering user")
	}
	logger.Info(ctx, "user registered", "user", userRegisterResponse.JSON200)

	return *userRegisterResponse.JSON200.UserId, nil
}

func userGet(ctx context.Context, c *clienthttp.ClientWithResponses, userId string) error {
	getUserIdResponse, err := c.GetUserGetIdWithResponse(ctx, userId)
	if err != nil {
		logger.Error(ctx, err, "failed getting user by id")
		return err
	}
	if getUserIdResponse.StatusCode() != 200 {
		logger.Warn(ctx, "failed getting user by id", "response", string(getUserIdResponse.Body), "status", getUserIdResponse.StatusCode())
		return errors.New("failed getting user by id")
	}
	logger.Info(ctx, "user got by id", "user", getUserIdResponse.JSON200)
	return nil
}

func login(ctx context.Context, c *clienthttp.ClientWithResponses, userId string, password string) (string, error) {
	loginResponse, err := c.PostLoginWithResponse(ctx, model.PostLoginJSONRequestBody{
		Id:       ptrString(userId),
		Password: ptrString(password),
	})
	if err != nil {
		logger.Error(ctx, err, "failed login")
		return "", err
	}
	if loginResponse.StatusCode() != 200 {
		logger.Warn(ctx, "failed login", "response", string(loginResponse.Body), "status", loginResponse.StatusCode())
		return "", errors.New("failed login")
	}
	logger.Info(ctx, "login succeeded", "user", loginResponse.JSON200)

	return *loginResponse.JSON200.Token, nil
}

func getMe(ctx context.Context, c *clienthttp.ClientWithResponses, addTokenFunc clienthttp.RequestEditorFn) error {
	getMeResponse, err := c.GetUserMeWithResponse(ctx, addTokenFunc)
	if err != nil {
		logger.Error(ctx, err, "failed get me")
		return err
	}
	if getMeResponse.StatusCode() != 200 {
		logger.Warn(ctx, "failed get me", "response", string(getMeResponse.Body), "status", getMeResponse.StatusCode())
		return errors.New("failed get me")
	}
	logger.Info(ctx, "get me succeeded", "user", getMeResponse.JSON200)
	return nil
}

func ptrString(s string) *string {
	return &s
}

func ptrGender(g model.Gender) *model.Gender {
	return &g
}
