package client

import (
	"errors"
	"strings"

	"github.com/epicoon/lxgo/kernel"
)

// ErrAuthMissing is returned by GetBearer when the request has no
// Authorization header at all.
var ErrAuthMissing = errors.New("authorization header missing")

// ErrAuthWrongScheme is returned by GetBearer when the Authorization header
// isn't a "Bearer <token>".
var ErrAuthWrongScheme = errors.New("invalid authorization scheme")

// GetBearer extracts the bearer token from ctx's Authorization header.
func GetBearer(ctx kernel.IHandleContext) (string, error) {
	authHeader := ctx.Request().Header.Get("Authorization")
	if authHeader == "" {
		return "", ErrAuthMissing
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", ErrAuthWrongScheme
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	return token, nil
}

// UserData is the data an authenticated user stored on the authorization
// service - see AuthClient.GetUserData.
type UserData struct {
	Login string
	Data  map[string]any
}
