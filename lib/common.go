package lib

const DefaultConfigDir = ".dead-drop"
const DefaultConfigName = "conf"
const TokenCipherLabel = "token"

type TokenRequestPayload struct {
	Key     []byte
	KeyName string
}
