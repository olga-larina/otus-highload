package internalhttp

import (
	"context"
	"errors"

	"github.com/olga-larina/otus-highload/backend/internal/logger"
	"github.com/olga-larina/otus-highload/backend/internal/model"
)

type Server struct {
	authService     AuthService
	loginService    LoginService
	userService     UserService
	friendService   FriendService
	postService     PostService
	postFeedService PostFeedService
}

type AuthService interface {
	GetUserId(ctx context.Context) (string, error)
}

type LoginService interface {
	Login(ctx context.Context, request *model.PostLoginJSONRequestBody) (string, error)
}

type UserService interface {
	GetUserById(ctx context.Context, userId *model.UserId) (*model.User, error)
	RegisterUser(ctx context.Context, registerBody *model.PostUserRegisterJSONRequestBody) (*model.UserId, error)
	SearchByName(ctx context.Context, firstNamePrefix string, lastNamePrefix string) ([]*model.User, error)
}

type FriendService interface {
	AddFriend(ctx context.Context, userId *model.UserId, friendId *model.UserId) error
	DeleteFriend(ctx context.Context, userId *model.UserId, friendId *model.UserId) error
}

type PostService interface {
	CreatePost(ctx context.Context, userId *model.UserId, postText *model.PostText) (*model.PostExtended, error)
	UpdatePost(ctx context.Context, userId *model.UserId, postId *model.PostId, postText *model.PostText) (*model.PostExtended, error)
	DeletePost(ctx context.Context, userId *model.UserId, postId *model.PostId) error
	GetPostById(ctx context.Context, userId *model.UserId, postId *model.PostId) (*model.PostExtended, error)
}

type PostFeedService interface {
	GetUserPosts(ctx context.Context, userId *model.UserId, limit int, offset int) ([]*model.PostExtended, error)
}

func NewServer(
	authService AuthService,
	loginService LoginService,
	userService UserService,
	friendService FriendService,
	postService PostService,
	postFeedService PostFeedService,
) *Server {
	return &Server{
		authService:     authService,
		loginService:    loginService,
		userService:     userService,
		friendService:   friendService,
		postService:     postService,
		postFeedService: postFeedService,
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
	_, err := s.authService.GetUserId(ctx)
	if err != nil {
		logger.Error(ctx, err, "not authorized")
		return GetUserGetId401Response{}, nil
	}
	user, err := s.userService.GetUserById(ctx, &request.Id)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			logger.Error(ctx, err, "failed to get user")
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
	userId, err := s.authService.GetUserId(ctx)
	if err != nil {
		logger.Error(ctx, err, "not authorized")
		return GetUserMe401Response{}, nil
	}
	user, err := s.userService.GetUserById(ctx, &userId)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			logger.Error(ctx, err, "failed to get me")
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
	_, err := s.authService.GetUserId(ctx)
	if err != nil {
		logger.Error(ctx, err, "not authorized")
		return GetUserSearch401Response{}, nil
	}
	users, err := s.userService.SearchByName(ctx, request.Params.FirstName, request.Params.LastName)
	if err != nil {
		response := GetUserSearch500JSONResponse{}
		response.Body.Message = err.Error()
		return response, nil
	}
	response := GetUserSearch200JSONResponse{}
	for _, user := range users {
		response = append(response, *user)
	}
	return response, nil
}

// (PUT /friend/set/{user_id})
func (s *Server) PutFriendSetUserId(ctx context.Context, request PutFriendSetUserIdRequestObject) (PutFriendSetUserIdResponseObject, error) {
	userId, err := s.authService.GetUserId(ctx)
	if err != nil {
		logger.Error(ctx, err, "not authorized")
		return PutFriendSetUserId401Response{}, nil
	}
	err = s.friendService.AddFriend(ctx, &userId, &request.UserId)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			logger.Error(ctx, err, "failed to add friend")
			return PutFriendSetUserId404Response{}, nil
		}
		if errors.Is(err, model.ErrUserAlreadyExists) {
			logger.Error(ctx, err, "failed to add friend")
			return PutFriendSetUserId400Response{}, nil
		}
		response := PutFriendSetUserId500JSONResponse{}
		response.Body.Message = err.Error()
		return response, nil
	}
	return PutFriendSetUserId200Response{}, nil
}

// (PUT /friend/delete/{user_id})
func (s *Server) PutFriendDeleteUserId(ctx context.Context, request PutFriendDeleteUserIdRequestObject) (PutFriendDeleteUserIdResponseObject, error) {
	userId, err := s.authService.GetUserId(ctx)
	if err != nil {
		logger.Error(ctx, err, "not authorized")
		return PutFriendDeleteUserId401Response{}, nil
	}
	err = s.friendService.DeleteFriend(ctx, &userId, &request.UserId)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			logger.Error(ctx, err, "failed to delete friend")
			return PutFriendDeleteUserId404Response{}, nil
		}
		response := PutFriendDeleteUserId500JSONResponse{}
		response.Body.Message = err.Error()
		return response, nil
	}
	return PutFriendDeleteUserId200Response{}, nil
}

// (POST /post/create)
func (s *Server) PostPostCreate(ctx context.Context, request PostPostCreateRequestObject) (PostPostCreateResponseObject, error) {
	userId, err := s.authService.GetUserId(ctx)
	if err != nil {
		logger.Error(ctx, err, "not authorized")
		return PostPostCreate401Response{}, nil
	}
	post, err := s.postService.CreatePost(ctx, &userId, &request.Body.Text)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			logger.Error(ctx, err, "failed to create post")
			return PostPostCreate404Response{}, nil
		}
		response := PostPostCreate500JSONResponse{}
		response.Body.Message = err.Error()
		return response, nil
	}
	return PostPostCreate200JSONResponse(*post.Id), nil
}

// (PUT /post/update)
func (s *Server) PutPostUpdate(ctx context.Context, request PutPostUpdateRequestObject) (PutPostUpdateResponseObject, error) {
	userId, err := s.authService.GetUserId(ctx)
	if err != nil {
		logger.Error(ctx, err, "not authorized")
		return PutPostDeleteId401Response{}, nil
	}
	_, err = s.postService.UpdatePost(ctx, &userId, &request.Body.Id, &request.Body.Text)
	if err != nil {
		if errors.Is(err, model.ErrPostNotFound) {
			logger.Error(ctx, err, "failed to update post")
			return PutPostUpdate404Response{}, nil
		}
		response := PutPostUpdate500JSONResponse{}
		response.Body.Message = err.Error()
		return response, nil
	}
	return PutPostUpdate200Response{}, nil
}

// (PUT /post/delete/{id})
func (s *Server) PutPostDeleteId(ctx context.Context, request PutPostDeleteIdRequestObject) (PutPostDeleteIdResponseObject, error) {
	userId, err := s.authService.GetUserId(ctx)
	if err != nil {
		logger.Error(ctx, err, "not authorized")
		return PutPostDeleteId401Response{}, nil
	}
	err = s.postService.DeletePost(ctx, &userId, &request.Id)
	if err != nil {
		if errors.Is(err, model.ErrPostNotFound) {
			logger.Error(ctx, err, "failed to delete post")
			return PutPostDeleteId404Response{}, nil
		}
		response := PutPostDeleteId500JSONResponse{}
		response.Body.Message = err.Error()
		return response, nil
	}
	return PutPostDeleteId200Response{}, nil
}

// (GET /post/get/{id})
func (s *Server) GetPostGetId(ctx context.Context, request GetPostGetIdRequestObject) (GetPostGetIdResponseObject, error) {
	userId, err := s.authService.GetUserId(ctx)
	if err != nil {
		logger.Error(ctx, err, "not authorized")
		return GetPostGetId401Response{}, nil
	}
	post, err := s.postService.GetPostById(ctx, &userId, &request.Id)
	if err != nil {
		if errors.Is(err, model.ErrPostNotFound) {
			logger.Error(ctx, err, "failed to get post")
			return GetPostGetId404Response{}, nil
		}
		response := GetPostGetId500JSONResponse{}
		response.Body.Message = err.Error()
		return response, nil
	}
	return GetPostGetId200JSONResponse(post.Post), nil
}

// (GET /post/feed)
func (s *Server) GetPostFeed(ctx context.Context, request GetPostFeedRequestObject) (GetPostFeedResponseObject, error) {
	userId, err := s.authService.GetUserId(ctx)
	if err != nil {
		logger.Error(ctx, err, "not authorized")
		return GetPostFeed401Response{}, nil
	}
	limit := int(*request.Params.Limit)
	offset := int(*request.Params.Offset)
	if limit <= 0 || offset < 0 {
		logger.Error(ctx, errors.New("not valid limit or offset"), "failed to get feed")
		return GetPostFeed400Response{}, nil
	}
	posts, err := s.postFeedService.GetUserPosts(ctx, &userId, limit, offset)
	if err != nil {
		if errors.Is(err, model.ErrPostFeedLenNotValid) {
			logger.Error(ctx, err, "failed to get feed")
			return GetPostFeed400Response{}, nil
		}
		response := GetPostFeed500JSONResponse{}
		response.Body.Message = err.Error()
		return response, nil
	}
	response := GetPostFeed200JSONResponse{}
	for _, post := range posts {
		response = append(response, post.Post)
	}
	return response, nil
}

// (GET /dialog/{user_id}/list)
func (s *Server) GetDialogUserIdList(ctx context.Context, request GetDialogUserIdListRequestObject) (GetDialogUserIdListResponseObject, error) {
	return GetDialogUserIdList400Response{}, nil
}

// (POST /dialog/{user_id}/send)
func (s *Server) PostDialogUserIdSend(ctx context.Context, request PostDialogUserIdSendRequestObject) (PostDialogUserIdSendResponseObject, error) {
	return PostDialogUserIdSend400Response{}, nil
}
