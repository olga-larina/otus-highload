package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

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

	withoutTokenFunc := func(ctx context.Context, req *http.Request) error { return nil }

	// --------- Проверка корректных запросов

	// регистрация пользователя
	userId, err = userRegister(ctx, c, password)
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

	// получение пользователя по ID
	err = getUser(ctx, c, addTokenFunc, userId)
	if err != nil {
		return
	}

	// получение информации о себе (через метод, требующий токена)
	err = getMe(ctx, c, addTokenFunc)
	if err != nil {
		return
	}

	// поиск пользователей
	err = searchUsers(ctx, c, addTokenFunc, "Конст", "Оси")
	if err != nil {
		return
	}

	// добавление друга
	// - регистрация ещё одного пользователя для этого
	friendId, err := userRegister(ctx, c, password)
	if err != nil {
		return
	}

	err = addFriend(ctx, c, addTokenFunc, friendId)
	if err != nil {
		return
	}

	// - получение токена друга
	tokenFriend, err := login(ctx, c, friendId, password)
	if err != nil {
		return
	}

	// - авторизация друга
	addTokenFriendFunc := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+tokenFriend)
		return nil
	}

	// добавление поста друга
	postId, err := createPost(ctx, c, addTokenFriendFunc)
	if err != nil {
		return
	}

	// обновление поста друга
	err = updatePost(ctx, c, addTokenFriendFunc, postId)
	if err != nil {
		return
	}

	// получение поста друга
	err = getPost(ctx, c, addTokenFriendFunc, postId)
	if err != nil {
		return
	}

	// получение ленты постов
	err = getFeed(ctx, c, addTokenFunc, 1000, 0)
	if err != nil {
		return
	}

	// удаление поста друга
	err = deletePost(ctx, c, addTokenFriendFunc, postId)
	if err != nil {
		return
	}

	// удаление друга
	err = deleteFriend(ctx, c, addTokenFunc, friendId)
	if err != nil {
		return
	}

	// добавление сообщения от пользователя 1 пользователю 2
	err = createDialogMessage(ctx, c, addTokenFunc, friendId)
	if err != nil {
		return
	}

	// добавление сообщения от пользователя 2 пользователю 1
	err = createDialogMessage(ctx, c, addTokenFriendFunc, userId)
	if err != nil {
		return
	}

	// получение диалога пользователем 1
	err = getDialog(ctx, c, addTokenFunc, friendId)
	if err != nil {
		return
	}

	// получение диалога пользователем 2
	err = getDialog(ctx, c, addTokenFriendFunc, userId)
	if err != nil {
		return
	}

	// --------- Проверка некорректных запросов

	// логин с некорректным паролем
	login(ctx, c, userId, "ABC")

	// несуществующий пользователь
	getUser(ctx, c, addTokenFunc, "ABC")

	// получение информации о себе без токена
	getMe(ctx, c, withoutTokenFunc)

	// поиск пользователей без токена
	searchUsers(ctx, c, withoutTokenFunc, "Конст", "Оси")

	// добавление несуществующего пользователя в друзья
	addFriend(ctx, c, addTokenFunc, "abc")

	// удаление несуществующего друга
	deleteFriend(ctx, c, addTokenFunc, "abc")

	// создание поста без токена
	createPost(ctx, c, withoutTokenFunc)

	// обновление несуществующего поста
	updatePost(ctx, c, addTokenFunc, "abc")

	// удаление несуществующего поста
	deletePost(ctx, c, addTokenFunc, "abc")

	// получение несуществующего поста
	getPost(ctx, c, addTokenFunc, "abc")

	// получение ленты без токена
	getFeed(ctx, c, withoutTokenFunc, 100, 0)

	// добавление сообщения без токена
	createDialogMessage(ctx, c, withoutTokenFunc, userId)

	// получение диалога без токена
	getDialog(ctx, c, withoutTokenFunc, friendId)

	logger.Info(ctx, "successfully finished")
}

func userRegister(ctx context.Context, c *clienthttp.ClientWithResponses, password string) (model.UserId, error) {
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

func getUser(ctx context.Context, c *clienthttp.ClientWithResponses, addTokenFunc clienthttp.RequestEditorFn, userId string) error {
	getUserIdResponse, err := c.GetUserGetIdWithResponse(ctx, userId, addTokenFunc)
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

func searchUsers(ctx context.Context, c *clienthttp.ClientWithResponses, addTokenFunc clienthttp.RequestEditorFn, firstName string, lastName string) error {
	usersSearchResponse, err := c.GetUserSearchWithResponse(ctx, &model.GetUserSearchParams{FirstName: firstName, LastName: lastName}, addTokenFunc)
	if err != nil {
		logger.Error(ctx, err, "failed searching users")
		return err
	}
	if usersSearchResponse.StatusCode() != 200 {
		logger.Warn(ctx, "failed searching users", "response", string(usersSearchResponse.Body), "status", usersSearchResponse.StatusCode())
		return errors.New("failed searching users")
	}
	logger.Info(ctx, "users found by names", "users", usersSearchResponse.JSON200)
	return nil
}

func addFriend(ctx context.Context, c *clienthttp.ClientWithResponses, addTokenFunc clienthttp.RequestEditorFn, friendId string) error {
	addFriendResponse, err := c.PutFriendSetUserIdWithResponse(ctx, friendId, addTokenFunc)
	if err != nil {
		logger.Error(ctx, err, "failed adding friend")
		return err
	}
	if addFriendResponse.StatusCode() != 200 {
		logger.Warn(ctx, "failed adding friend", "response", string(addFriendResponse.Body), "status", addFriendResponse.StatusCode())
		return errors.New("failed adding friend")
	}
	logger.Info(ctx, "friend added")

	return nil
}

func deleteFriend(ctx context.Context, c *clienthttp.ClientWithResponses, addTokenFunc clienthttp.RequestEditorFn, friendId string) error {
	deleteFriendResponse, err := c.PutFriendDeleteUserIdWithResponse(ctx, friendId, addTokenFunc)
	if err != nil {
		logger.Error(ctx, err, "failed deleting friend")
		return err
	}
	if deleteFriendResponse.StatusCode() != 200 {
		logger.Warn(ctx, "failed deleting friend", "response", string(deleteFriendResponse.Body), "status", deleteFriendResponse.StatusCode())
		return errors.New("failed deleting friend")
	}
	logger.Info(ctx, "friend deleted")

	return nil
}

func createPost(ctx context.Context, c *clienthttp.ClientWithResponses, addTokenFunc clienthttp.RequestEditorFn) (model.PostId, error) {
	createPostResponse, err := c.PostPostCreateWithResponse(
		ctx,
		model.PostPostCreateJSONRequestBody{Text: fmt.Sprintf("test %v", time.Now())},
		addTokenFunc,
	)
	if err != nil {
		logger.Error(ctx, err, "failed creating post")
		return "", err
	}
	if createPostResponse.StatusCode() != 200 {
		logger.Warn(ctx, "failed creating post", "response", string(createPostResponse.Body), "status", createPostResponse.StatusCode())
		return "", errors.New("failed creating post")
	}
	logger.Info(ctx, "post created")

	return *createPostResponse.JSON200, nil
}

func updatePost(ctx context.Context, c *clienthttp.ClientWithResponses, addTokenFunc clienthttp.RequestEditorFn, postId model.PostId) error {
	updatePostResponse, err := c.PutPostUpdateWithResponse(
		ctx,
		model.PutPostUpdateJSONRequestBody{Id: postId, Text: fmt.Sprintf("test2 %v", time.Now())},
		addTokenFunc,
	)
	if err != nil {
		logger.Error(ctx, err, "failed updating post")
		return err
	}
	if updatePostResponse.StatusCode() != 200 {
		logger.Warn(ctx, "failed updating post", "response", string(updatePostResponse.Body), "status", updatePostResponse.StatusCode())
		return errors.New("failed updating post")
	}
	logger.Info(ctx, "post updated")

	return nil
}

func deletePost(ctx context.Context, c *clienthttp.ClientWithResponses, addTokenFunc clienthttp.RequestEditorFn, postId model.PostId) error {
	deletePostResponse, err := c.PutPostDeleteIdWithResponse(ctx, postId, addTokenFunc)
	if err != nil {
		logger.Error(ctx, err, "failed deleting post")
		return err
	}
	if deletePostResponse.StatusCode() != 200 {
		logger.Warn(ctx, "failed deleting post", "response", string(deletePostResponse.Body), "status", deletePostResponse.StatusCode())
		return errors.New("failed deleting post")
	}
	logger.Info(ctx, "post deleted")

	return nil
}

func getPost(ctx context.Context, c *clienthttp.ClientWithResponses, addTokenFunc clienthttp.RequestEditorFn, postId model.PostId) error {
	getPostResponse, err := c.GetPostGetIdWithResponse(ctx, postId, addTokenFunc)
	if err != nil {
		logger.Error(ctx, err, "failed getting post")
		return err
	}
	if getPostResponse.StatusCode() != 200 {
		logger.Warn(ctx, "failed getting post", "response", string(getPostResponse.Body), "status", getPostResponse.StatusCode())
		return errors.New("failed getting post")
	}
	logger.Info(ctx, "post obtained", "post", getPostResponse.JSON200)

	return nil
}

func getFeed(ctx context.Context, c *clienthttp.ClientWithResponses, addTokenFunc clienthttp.RequestEditorFn, limit int, offset int) error {
	limitFloat := float32(limit)
	offsetFloat := float32(offset)
	getFeedResponse, err := c.GetPostFeedWithResponse(ctx, &model.GetPostFeedParams{Limit: &limitFloat, Offset: &offsetFloat}, addTokenFunc)
	if err != nil {
		logger.Error(ctx, err, "failed getting feed")
		return err
	}
	if getFeedResponse.StatusCode() != 200 {
		logger.Warn(ctx, "failed getting feed", "response", string(getFeedResponse.Body), "status", getFeedResponse.StatusCode())
		return errors.New("failed getting feed")
	}
	logger.Info(ctx, "feed obtained", "feed", getFeedResponse.JSON200)

	return nil
}

func createDialogMessage(ctx context.Context, c *clienthttp.ClientWithResponses, addTokenFunc clienthttp.RequestEditorFn, toUserId model.UserId) error {
	createDialogMessageResponse, err := c.PostDialogUserIdSendWithResponse(
		ctx,
		toUserId,
		model.PostDialogUserIdSendJSONRequestBody{Text: fmt.Sprintf("test message %v", time.Now())},
		addTokenFunc,
	)
	if err != nil {
		logger.Error(ctx, err, "failed creating dialog message")
		return err
	}
	if createDialogMessageResponse.StatusCode() != 200 {
		logger.Warn(ctx, "failed creating dialog message", "response", string(createDialogMessageResponse.Body), "status", createDialogMessageResponse.StatusCode())
		return errors.New("failed creating dialog message")
	}
	logger.Info(ctx, "dialog message created")

	return nil
}

func getDialog(ctx context.Context, c *clienthttp.ClientWithResponses, addTokenFunc clienthttp.RequestEditorFn, withUserId model.UserId) error {
	getDialogResponse, err := c.GetDialogUserIdListWithResponse(ctx, withUserId, addTokenFunc)
	if err != nil {
		logger.Error(ctx, err, "failed getting dialog")
		return err
	}
	if getDialogResponse.StatusCode() != 200 {
		logger.Warn(ctx, "failed getting dialog", "response", string(getDialogResponse.Body), "status", getDialogResponse.StatusCode())
		return errors.New("failed getting dialog")
	}
	logger.Info(ctx, "dialog obtained", "dialog", getDialogResponse.JSON200)

	return nil
}

func ptrString(s string) *string {
	return &s
}

func ptrGender(g model.Gender) *model.Gender {
	return &g
}
