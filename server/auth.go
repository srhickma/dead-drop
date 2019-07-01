package main

import (
	"crypto/rand"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/logger"
	"time"
)

const secret = "TODO PUT A RANDOM SECRET HERE!!!!!"

func generateToken() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims {
		"ran": randomClaim(),
		"exp": time.Now().Add(time.Second).Unix(),
	})

	return token.SignedString([]byte(secret))
}

func validateToken(tokenString string) bool {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})
	if err != nil {
		return false
	}

	_, ok := token.Claims.(jwt.MapClaims)
	return ok && token.Valid
}

func randomClaim() string {
	const characters = "abcdefghijklmnopqrstuvwxyz"
	const length = 32

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		logger.Fatalf("Failed to generate random token claim: %v\n", err)
	}

	modulo := byte(len(characters))
	for i, b := range bytes {
		bytes[i] = characters[b%modulo]
	}
	return string(bytes)
}