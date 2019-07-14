package lib

const DefaultConfigDir = ".dead-drop"
const DefaultConfigName = "conf"
const TokenCipherLabel = "token"
const KeyNameRegex = ""

type TokenRequestPayload struct {
	KeyName string
}
