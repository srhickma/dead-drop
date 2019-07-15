package lib

const ObjectPerms = 0660
const PrivateKeyPerms = 0600
const PublicKeyPerms = 0660
const DefaultConfigDir = ".dead-drop"
const DefaultConfigName = "conf"
const TokenCipherLabel = "token"
const KeyNameRegex = "^[a-zA-Z0-9_-]{1,64}$"

type TokenRequestPayload struct {
	KeyName string
}

type AddKeyPayload struct {
	Key     []byte
	KeyName string
}
