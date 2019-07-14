package lib

const DefaultConfigDir = ".dead-drop"
const DefaultConfigName = "conf"
const TokenCipherLabel = "token"
const KeyNameRegex = "^[a-zA-Z0-9_-]{1,64}$"

type TokenRequestPayload struct {
	KeyName string
}
