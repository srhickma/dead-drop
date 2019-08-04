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
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"
)

const UnauthorizedErr = Error("math: square root of negative number")

type Authenticator struct {
	secret            []byte
	secretLock        sync.RWMutex
	authorizedKeysDir string
}

func newAuthenticator(authorizedKeysDirPath string) *Authenticator {
	authorizedKeysDir, err := homedir.Expand(authorizedKeysDirPath)
	if err != nil {
		logger.Fatalf("Failed to expand authorized keys file path: %v", err)
	}

	logger.Infof("Starting authenticator with authorized-keys directory %s", authorizedKeysDir)

	authenticator := &Authenticator{
		secret:            newSecret(),
		authorizedKeysDir: authorizedKeysDir,
	}

	go authenticator.secretRotator()

	return authenticator
}

func (auth *Authenticator) secretRotator() {
	const rotationSeconds = 16

	for {
		time.Sleep(rotationSeconds * time.Second)

		auth.secretLock.Lock()
		auth.secret = newSecret()
		auth.secretLock.Unlock()
	}
}

func (auth *Authenticator) generateToken(pkeyBytes []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"ran": auth.randomClaim(),
		"exp": time.Now().Add(time.Second).Unix(),
	})

	auth.secretLock.RLock()
	signedToken, err := token.SignedString(auth.secret)
	if err != nil {
		return "", err
	}
	auth.secretLock.RUnlock()

	pkeyDer, _ := pem.Decode(pkeyBytes)
	if pkeyDer == nil {
		logger.Errorf("Failed to decode pem bytes")
		return "", UnauthorizedErr
	}
	pkey, err := x509.ParsePKCS1PublicKey(pkeyDer.Bytes)
	if err != nil {
		logger.Errorf("Failed to parse public key: %v", err)
		return "", UnauthorizedErr
	}

	ciphertext, err := rsa.EncryptOAEP(sha512.New(), rand.Reader, pkey, []byte(signedToken), []byte(lib.TokenCipherLabel))
	return string(ciphertext), err
}

func (auth *Authenticator) validateToken(tokenString string) bool {
	auth.secretLock.RLock()
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return auth.secret, nil
	})
	auth.secretLock.RUnlock()
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
		logger.Fatalf("Failed to generate random token claim: %v", err)
	}

	modulo := byte(len(characters))
	for i, b := range bytes {
		bytes[i] = characters[b%modulo]
	}
	return string(bytes)
}

func (auth *Authenticator) getAuthorizedKey(keyName string) ([]byte, error) {
	return ioutil.ReadFile(filepath.Join(auth.authorizedKeysDir, keyName))
}

func (auth *Authenticator) addAuthorizedKey(key []byte, keyName string) error {
	return ioutil.WriteFile(filepath.Join(auth.authorizedKeysDir, keyName), key, lib.PublicKeyPerms)
}

func newSecret() []byte {
	const length = 64

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		logger.Fatalf("Failed to generate random secret: %v", err)
	}

	return bytes
}
