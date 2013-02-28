package identityservice

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

type UserInfo struct {
	Id       string
	TenantId string
	Token    string
	secret   string
}

var randReader = rand.Reader

// Change how we get random data, the default is to use crypto/rand
// This mostly exists to be able to test error side effects
// The return value is a function you can call to restore the previous
// randomizer
// Note: This is not thread safe, but it is really only for testing anyway
func setReader(r io.Reader) (restore func()) {
	old := randReader
	randReader = r
	return func() {
		randReader = old
	}
}

// Generate a bit of random hex data for
func randomHexToken() string {
	raw_bytes := make([]byte, 16)
	n, err := io.ReadFull(randReader, raw_bytes)
	if n != 16 || err != nil {
		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		panic(fmt.Sprintf(
			"Could not read 16 random bytes safely (read %d bytes): %s",
			n, errStr))
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
