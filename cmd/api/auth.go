package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthClaims struct {
	Subject string
	Role    string
	Expiry  time.Time
}

type customClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func (app *application) generateToken(subject, role string, ttl time.Duration) (string, error) {
	claims := customClaims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    app.config.security.tokenIssuer,
			Audience:  jwt.ClaimStrings{app.config.security.tokenAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(app.config.security.tokenSecret))
}

func (app *application) parseToken(tokenString string) (AuthClaims, error) {
	claims := customClaims{}

	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(app.config.security.tokenSecret), nil
	}, jwt.WithIssuer(app.config.security.tokenIssuer), jwt.WithAudience(app.config.security.tokenAudience))
	if err != nil {
		return AuthClaims{}, err
	}

	if !token.Valid {
		return AuthClaims{}, errors.New("invalid token")
	}

	return AuthClaims{
		Subject: claims.Subject,
		Role:    claims.Role,
		Expiry:  claims.ExpiresAt.Time,
	}, nil
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader == "" {
			app.unauthorizedResponse(w, r)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			app.unauthorizedResponse(w, r)
			return
		}

		claims, err := app.parseToken(parts[1])
		if err != nil {
			app.unauthorizedResponse(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), userRoleContextKey, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) requireRoles(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(userRoleContextKey).(string)
			if _, ok := allowed[role]; !ok {
				app.forbiddenResponse(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
