package grpc

import (
	"errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
)

var TokenRequiredError = errors.New("token is required")
var InvalidTokenError = errors.New("invalid token")

type Authorizer interface {
	Authorize(ctx context.Context) error
}

type TokenAuthorizer struct {
	valid map[string]struct{}
}

var _ Authorizer = &TokenAuthorizer{}

func NewTokenAuthorizer(validTokens []string) *TokenAuthorizer {
	m := make(map[string]struct{})
	for _, t := range validTokens {
		m[t] = struct{}{}
	}
	return &TokenAuthorizer{valid: m}
}

func (t *TokenAuthorizer) Authorize(ctx context.Context) error {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		return TokenRequiredError
	}

	tokens := md[MetadataKeyToken]
	if len(tokens) != 1 || tokens[0] == "" {
		return TokenRequiredError
	}
	_, found := t.valid[tokens[0]]
	if !found {
		return InvalidTokenError
	}
	return nil
}
