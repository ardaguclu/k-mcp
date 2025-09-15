/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mcp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

// JWTClaims represents the claims in our JWT tokens.
// In a real application, you would include additional claims like issuer, audience, etc.
type JWTClaims struct {
	Scopes []string `json:"scopes"`
	jwt.RegisteredClaims
}

func verifyJWT(ctx context.Context, tokenString string, _ *http.Request) (*auth.TokenInfo, error) {
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenString, &JWTClaims{})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse token: %v", auth.ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("%w: invalid token", auth.ErrInvalidToken)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("%w: invalid token claims", auth.ErrInvalidToken)
	}

	if claims.ExpiresAt == nil {
		return nil, fmt.Errorf("%w: invalid token expired", auth.ErrInvalidToken)
	}

	if claims.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("%w: token has expired", auth.ErrInvalidToken)
	}

	if claims.NotBefore != nil && claims.NotBefore.After(time.Now()) {
		return nil, fmt.Errorf("%w: token not yet valid", auth.ErrInvalidToken)
	}

	if claims.Audience == nil {
		return nil, fmt.Errorf("%w: invalid token audience", auth.ErrInvalidToken)
	}

	found := false
	for _, aud := range claims.Audience {
		if aud == "k-mcp" {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("%w: token audience does not match k-mcp", auth.ErrInvalidToken)
	}

	return &auth.TokenInfo{
		Scopes:     claims.Scopes,
		Expiration: claims.ExpiresAt.Time,
	}, nil
}
