package internalhttp

import (
	"context"
	"errors"

	"github.com/olga-larina/otus-highload/backend/internal/logger"
	"github.com/olga-larina/otus-highload/backend/internal/model"
)

type Server struct {
	loginService LoginService
	userService  UserService
}

type LoginService interface {
	Login(ctx context.Context, request *model.PostLoginJSONRequestBody) (string, error)
}

type UserService interface {
	GetMe(ctx context.Context) (*model.User, error)
	GetUserById(ctx context.Context, id *model.UserId) (*model.User, error)
	RegisterUser(ctx context.Context, registerBody *model.PostUserRegisterJSONRequestBody) (*model.UserId, error)
}

func NewServer(
	loginService LoginService,
	userService UserService,
) *Server {
	return &Server{
		loginService: loginService,
		userService:  userService,
	}
}

// (POST /login)
func (s *Server) PostLogin(ctx context.Context, request PostLoginRequestObject) (PostLoginResponseObject, error) {
	token, err := s.loginService.Login(ctx, request.Body)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			logger.Error(ctx, err, "failed to login")
			return PostLogin404Response{}, nil
		}
		if errors.Is(err, model.ErrNotValidCredentials) {
			logger.Error(ctx, err, "failed to login")
			return PostLogin400Response{}, nil
		}
		response := PostLogin500JSONResponse{}
		response.Body.Message = err.Error()
		return response, nil
	}
	if len(token) == 0 {
		return PostLogin500JSONResponse{}, err
	}
	return PostLogin200JSONResponse{Token: &token}, nil
}

// (GET /user/get/{id})
func (s *Server) GetUserGetId(ctx context.Context, request GetUserGetIdRequestObject) (GetUserGetIdResponseObject, error) {
	user, err := s.userService.GetUserById(ctx, &request.Id)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return GetUserGetId404Response{}, nil
		}
		response := GetUserGetId500JSONResponse{}
		response.Body.Message = err.Error()
		return response, nil
	}
	return GetUserGetId200JSONResponse(*user), nil
}

// (POST /user/register)
func (s *Server) PostUserRegister(ctx context.Context, request PostUserRegisterRequestObject) (PostUserRegisterResponseObject, error) {
	userId, err := s.userService.RegisterUser(ctx, request.Body)
	if err != nil {
		if errors.Is(err, model.ErrNotValidPassword) {
			logger.Error(ctx, err, "failed to register user")
			return PostUserRegister400Response{}, nil
		}
		response := PostUserRegister500JSONResponse{}
		response.Body.Message = err.Error()
		return response, nil
	}
	return PostUserRegister200JSONResponse{UserId: userId}, nil
}

// (GET /user/me)
func (s *Server) GetUserMe(ctx context.Context, request GetUserMeRequestObject) (GetUserMeResponseObject, error) {
	user, err := s.userService.GetMe(ctx)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return GetUserMe404Response{}, nil
		}
		response := GetUserMe500JSONResponse{}
		response.Body.Message = err.Error()
		return response, nil
	}
	return GetUserMe200JSONResponse(*user), nil
}

// (GET /user/search)
func (s *Server) GetUserSearch(ctx context.Context, request GetUserSearchRequestObject) (GetUserSearchResponseObject, error) {
	return GetUserSearch400Response{}, nil
}

// (GET /dialog/{user_id}/list)
func (s *Server) GetDialogUserIdList(ctx context.Context, request GetDialogUserIdListRequestObject) (GetDialogUserIdListResponseObject, error) {
	return GetDialogUserIdList400Response{}, nil
}

// (POST /dialog/{user_id}/send)
func (s *Server) PostDialogUserIdSend(ctx context.Context, request PostDialogUserIdSendRequestObject) (PostDialogUserIdSendResponseObject, error) {
	return PostDialogUserIdSend400Response{}, nil
}

// (PUT /friend/delete/{user_id})
func (s *Server) PutFriendDeleteUserId(ctx context.Context, request PutFriendDeleteUserIdRequestObject) (PutFriendDeleteUserIdResponseObject, error) {
	return PutFriendDeleteUserId400Response{}, nil
}

// (PUT /friend/set/{user_id})
func (s *Server) PutFriendSetUserId(ctx context.Context, request PutFriendSetUserIdRequestObject) (PutFriendSetUserIdResponseObject, error) {
	return PutFriendSetUserId400Response{}, nil
}

// (POST /post/create)
func (s *Server) PostPostCreate(ctx context.Context, request PostPostCreateRequestObject) (PostPostCreateResponseObject, error) {
	return PostPostCreate400Response{}, nil
}

// (PUT /post/delete/{id})
func (s *Server) PutPostDeleteId(ctx context.Context, request PutPostDeleteIdRequestObject) (PutPostDeleteIdResponseObject, error) {
	return PutPostDeleteId400Response{}, nil
}

// (GET /post/feed)
func (s *Server) GetPostFeed(ctx context.Context, request GetPostFeedRequestObject) (GetPostFeedResponseObject, error) {
	return GetPostFeed400Response{}, nil
}

// (GET /post/get/{id})
func (s *Server) GetPostGetId(ctx context.Context, request GetPostGetIdRequestObject) (GetPostGetIdResponseObject, error) {
	return GetPostGetId400Response{}, nil
}

// (PUT /post/update)
func (s *Server) PutPostUpdate(ctx context.Context, request PutPostUpdateRequestObject) (PutPostUpdateResponseObject, error) {
	return PutPostUpdate400Response{}, nil
}
