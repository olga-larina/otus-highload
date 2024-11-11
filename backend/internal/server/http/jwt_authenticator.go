package internalhttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/olga-larina/otus-highload/backend/internal/logger"
	"github.com/olga-larina/otus-highload/backend/internal/model"
)

type JWSValidator interface {
	ValidateJWS(jws string) (jwt.Token, error)
}

type TokenService interface {
	CheckTokenClaims(expectedClaims []string, t jwt.Token) error
	ExtractUserId(t jwt.Token) (string, error)
}

var (
	ErrNoAuthHeader      = errors.New("authorization header is missing")
	ErrInvalidAuthHeader = errors.New("authorization header is malformed")
)

func NewAuthenticator(v JWSValidator, t TokenService) openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		if err := Authenticate(ctx, input, v, t); err != nil {
			logger.Error(ctx, err, "failed to login")
			return err
		}
		return nil
	}
}

// Authenticate uses the specified validator to ensure a JWT is valid, then makes
// sure that the claims provided by the JWT match the scopes as required in the API.
func Authenticate(ctx context.Context, input *openapi3filter.AuthenticationInput, v JWSValidator, t TokenService) error {
	// Our security scheme is named BearerAuth, ensure this is the case
	if input.SecuritySchemeName != "bearerAuth" {
		return fmt.Errorf("security scheme %s != 'bearerAuth'", input.SecuritySchemeName)
	}

	// Now, we need to get the JWS from the request, to match the request expectations
	// against request contents.
	jws, err := GetJWSFromRequest(input.RequestValidationInput.Request)
	if err != nil {
		return fmt.Errorf("getting jws: %w", err)
	}

	// if the JWS is valid, we have a JWT, which will contain a bunch of claims.
	token, err := v.ValidateJWS(jws)
	if err != nil {
		return fmt.Errorf("validating JWS: %w", err)
	}

	// We've got a valid token now, and we can look into its claims to see whether
	// they match. Every single scope must be present in the claims.
	err = t.CheckTokenClaims(input.Scopes, token)
	if err != nil {
		return fmt.Errorf("token claims don't match: %w", err)
	}

	userId, err := t.ExtractUserId(token)
	if err != nil {
		return fmt.Errorf("token doesn't contain claim: %w", err)
	}

	// добавляем в контекст userId запроса
	*input.RequestValidationInput.Request = *input.RequestValidationInput.Request.WithContext(
		context.WithValue(input.RequestValidationInput.Request.Context(), model.UserIdContextKey, userId),
	)

	return nil
}

// GetJWSFromRequest extracts a JWS string from an Authorization: Bearer <jws> header
func GetJWSFromRequest(req *http.Request) (string, error) {
	authHdr := req.Header.Get("Authorization")
	if authHdr == "" {
		return "", ErrNoAuthHeader
	}
	// We expect a header value of the form "Bearer <token>", with 1 space after Bearer, per spec.
	prefix := "Bearer "
	if !strings.HasPrefix(authHdr, prefix) {
		return "", ErrInvalidAuthHeader
	}
	return authHdr[len(prefix):], nil
}
