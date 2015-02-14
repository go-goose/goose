package client_test

import (
	"flag"
	"testing"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/identity"
)

var live = flag.Bool("live", false, "Include live OpenStack (Canonistack) tests")
var liveAuthMode = flag.String(
	"live-auth-mode", "userpass", "The authentication mode to use when running live tests [all|legacy|userpass|keypair]")

func Test(t *testing.T) {
	var allAuthModes = []identity.AuthMode{identity.AuthLegacy, identity.AuthUserPass, identity.AuthKeyPair}
	var liveAuthModes []identity.AuthMode
	switch *liveAuthMode {
	default:
		t.Fatalf("Invalid auth method specified: %s", *liveAuthMode)
	case "all":
		liveAuthModes = allAuthModes
	case "":
	case "keypair":
		liveAuthModes = []identity.AuthMode{identity.AuthKeyPair}
	case "userpass":
		liveAuthModes = []identity.AuthMode{identity.AuthUserPass}
	case "legacy":
		liveAuthModes = []identity.AuthMode{identity.AuthLegacy}
	}

	if *live {
		cred, err := identity.CompleteCredentialsFromEnv()
		if err != nil {
			t.Fatalf("Error setting up test suite: %v", err)
		}
		registerOpenStackTests(cred, liveAuthModes)
	}
	registerLocalTests(allAuthModes)
	gc.TestingT(t)
}
