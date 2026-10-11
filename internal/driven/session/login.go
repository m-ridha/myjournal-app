package session

import (
	provider "github/mridha/myjournal-app/internal/core/connection"
)

func EncodeLoginFlow(secret string, flow provider.LoginFlow) (string, error) {
	return encode(secret, flow, []byte("oauth-login-v1"))
}

func DecodeLoginFlow(secret, value string) (provider.LoginFlow, error) {
	var flow provider.LoginFlow
	err := decode(secret, value, []byte("oauth-login-v1"), &flow)
	return flow, err
}
