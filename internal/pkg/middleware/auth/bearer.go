package auth

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/transport"
)

func BearerTokenFromAuthorizationHeader(authorization string) (string, bool) {
	if authorization == "" {
		return "", false
	}
	const prefix = "Bearer "
	if strings.HasPrefix(authorization, prefix) {
		token := strings.TrimSpace(strings.TrimPrefix(authorization, prefix))
		return token, token != ""
	}
	return "", false
}

func BearerTokenFromContext(ctx context.Context) (string, bool) {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return "", false
	}
	token, ok := BearerTokenFromAuthorizationHeader(tr.RequestHeader().Get("Authorization"))
	if ok {
		return token, true
	}
	return BearerTokenFromAuthorizationHeader(tr.RequestHeader().Get("authorization"))
}
