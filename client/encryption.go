package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

const ivLength = aes.BlockSize

func encrypt(key []byte, data []byte) ([]byte, error) {
	encryptionKey, hmacKey := splitKeyHash(key)

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, ivLength+len(data))
	iv := ciphertext[:ivLength]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext[ivLength:], data)

	hash := hmac.New(sha256.New, []byte(hmacKey))
	hash.Write([]byte(ciphertext))
	signature := hash.Sum(nil)

	message := append(signature, ciphertext...)
	return message, nil
}

func decrypt(key []byte, message []byte) ([]byte, error) {
	encryptionKey, hmacKey := splitKeyHash(key)

	signature := message[:sha256.Size]
	ciphertext := message[sha256.Size:]

	hash := hmac.New(sha256.New, []byte(hmacKey))
	hash.Write([]byte(ciphertext))
	expectedSignature := hash.Sum(nil)

	if !hmac.Equal(signature, expectedSignature) {
		return nil, fmt.Errorf("bad signature")
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}

	data := make([]byte, len(ciphertext)-ivLength)
	iv := ciphertext[:ivLength]
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(data, ciphertext[ivLength:])

	return data, nil
}

func splitKeyHash(key []byte) ([]byte, []byte) {
	sum := sha256.Sum256(key)
	return sum[:16], sum[16:]
}
