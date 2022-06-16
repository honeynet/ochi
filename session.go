package main

import (
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

func NewToken(secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{})
	return token.SignedString([]byte(secret))
}

func ValidateToken(tokenString string, secret string) (bool, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})

	return token.Valid, err
}
