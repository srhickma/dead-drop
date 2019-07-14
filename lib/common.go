package lib

const TokenCipherLabel = "token"

type TokenRequestPayload struct {
	Key     []byte
	KeyName string
}
