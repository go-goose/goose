package identityservice

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type UserInfo struct {
	secret string
	token  string
}

// Generate a bit of random hex data for 
func randomHexToken() string {
	raw_bytes := make([]byte, 16)
	n, err := rand.Read(raw_bytes)
	if n != 16 || err != nil {
		panic(fmt.Sprintf(
			"Could not read 16 random bytes safely: %d %s",
			n, err.Error()))
	}
	hex_bytes := make([]byte, 32)
	n = hex.Encode(hex_bytes, raw_bytes)
	if n != 32 || err != nil {
		panic(fmt.Sprintf(
			"Failed to Encode 32 bytes: %d %s",
			n, err.Error()))
	}
	return string(hex_bytes)
}
