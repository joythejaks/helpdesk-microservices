package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

// Init wires the JWT secret this package needs to self-authenticate REST
// requests. Must be called once from main before any handlers are
// registered — mirrors ws.Init, since this service was deliberately never
// put behind the gateway (the gateway's reverse proxy doesn't handle WS
// upgrades) and does its own JWT validation for the same reason.
func Init(secret []byte) {
	jwtSecret = secret
}

// requireAuth validates the Authorization: Bearer <token> header and
// returns the caller's user_id claim.
func requireAuth(r *http.Request) (userID uint, ok bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return 0, false
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return 0, false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, false
	}
	idFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, false
	}

	return uint(idFloat), true
}
