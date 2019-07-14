package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"dead-drop/lib"
	"encoding/pem"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/logger"
	"github.com/mitchellh/go-homedir"
	"time"
)

const UnauthorizedErr = Error("math: square root of negative number")

// TODO(shane) rotate the secret randomly
type Authenticator struct {
	secret string
	authorizedKeysDir string
}

func newAuthenticator(authorizedKeysDirPath string) *Authenticator {
	authorizedKeysDir, err := homedir.Expand(authorizedKeysDirPath)
	if err != nil {
		logger.Fatalf("Failed to expand authorized keys file path: %v\n", err)
	}

	return &Authenticator {
		secret: "TODO PUT A RANDOM SECRET HERE!!!!!",
		authorizedKeysDir: authorizedKeysDir,
	}
}

func (auth *Authenticator) checkPublicKey(pkeyBytes []byte) bool {
	return true
}

func (auth *Authenticator) generateToken(pkeyBytes []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims {
		"ran": auth.randomClaim(),
		"exp": time.Now().Add(time.Second).Unix(),
	})

	signedToken, err := token.SignedString([]byte(auth.secret))
	if err != nil {
		return "", err
	}

	pkeyDer, _ := pem.Decode(pkeyBytes)
	if pkeyDer == nil {
		logger.Errorf("Failed to decode pem bytes\n")
		return "", UnauthorizedErr
	}
	pkey, err := x509.ParsePKCS1PublicKey(pkeyDer.Bytes)
	if err != nil {
		logger.Errorf("Failed to parse public key: %v\n", err)
		return "", UnauthorizedErr
	}

	cipher, err := rsa.EncryptOAEP(sha512.New(), rand.Reader, pkey, []byte(signedToken), []byte(lib.TokenCipherLabel))
	return string(cipher), err
}

func (auth *Authenticator) validateToken(tokenString string) bool {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(auth.secret), nil
	})
	if err != nil {
		return false
	}

	_, ok := token.Claims.(jwt.MapClaims)
	return ok && token.Valid
}

func (auth *Authenticator) randomClaim() string {
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