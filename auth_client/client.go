package client

import (
	"errors"
	"strings"

	"github.com/epicoon/lxgo/kernel"
)

var ErrAuthMissing = errors.New("authorization header missing")
var ErrAuthWrongScheme = errors.New("invalid authorization scheme")

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

type UserData struct {
	Login string
	Data  map[string]any
}
