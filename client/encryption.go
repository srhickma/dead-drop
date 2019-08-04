package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"dead-drop/client/ghash"
	"fmt"
	"github.com/awnumar/memguard"
)

const ivLength = aes.BlockSize

func encrypt(key *memguard.LockedBuffer, data []byte) ([]byte, error) {
	encryptionKey, hmacKey := splitKeyHash(key)

	block, err := aes.NewCipher(encryptionKey.Bytes())
	if err != nil {
		return nil, err
	}

	ciphertext := memguard.NewBufferRandom(ivLength + len(data))
	defer ciphertext.Destroy()

	iv := ciphertext.Bytes()[:ivLength]
	stream := cipher.NewCTR(block, iv)

	ciphertext.Melt()
	stream.XORKeyStream(ciphertext.Bytes()[ivLength:], data)
	ciphertext.Freeze()

	encryptionKey.Destroy()

	hash := hmac.New(sha256.New, hmacKey.Bytes())
	hash.Write(ciphertext.Bytes())
	signature := hash.Sum(nil)
	hmacKey.Destroy()

	message := append(signature, ciphertext.Bytes()...)
	return message, nil
}

func decrypt(key *memguard.LockedBuffer, message []byte) (*memguard.LockedBuffer, error) {
	encryptionKey, hmacKey := splitKeyHash(key)

	signature := message[:sha256.Size]
	ciphertext := message[sha256.Size:]

	hash := hmac.New(sha256.New, hmacKey.Bytes())
	hash.Write([]byte(ciphertext))
	expectedSignature := hash.Sum(nil)
	hmacKey.Destroy()

	if !hmac.Equal(signature, expectedSignature) {
		return nil, fmt.Errorf("bad signature")
	}

	block, err := aes.NewCipher(encryptionKey.Bytes())
	if err != nil {
		return nil, err
	}

	data := memguard.NewBuffer(len(ciphertext) - ivLength)

	iv := ciphertext[:ivLength]
	stream := cipher.NewCTR(block, iv)

	data.Melt()
	stream.XORKeyStream(data.Bytes(), ciphertext[ivLength:])
	data.Freeze()

	encryptionKey.Destroy()

	return data, nil
}

func splitKeyHash(keyBuf *memguard.LockedBuffer) (*memguard.LockedBuffer, *memguard.LockedBuffer) {
	sum := ghash.Sum256(keyBuf)
	defer sum.Destroy()

	keyBuf.Destroy()

	sum.Melt()
	key1 := memguard.NewBufferFromBytes(sum.Bytes()[:16])
	key2 := memguard.NewBufferFromBytes(sum.Bytes()[16:])

	return key1, key2
}
