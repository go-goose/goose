package client_test

import (
	"flag"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/identity"
	"testing"
)

var live = flag.Bool("live", false, "Include live OpenStack (Canonistack) tests")
var liveAuthMode = flag.String(
	"live-auth-mode", "userpass", "The authentication mode to use when running live tests [all|legacy|userpass]")

func Test(t *testing.T) {
	var allAuthModes = []identity.AuthMode{identity.AuthLegacy, identity.AuthUserPass}
	var liveAuthModes []identity.AuthMode
	switch *liveAuthMode {
	default:
		t.Fatalf("Invalid auth method specified: %s", *liveAuthMode)
	case "all":
		liveAuthModes = allAuthModes
	case "":
	case "userpass":
		liveAuthModes = []identity.AuthMode{identity.AuthUserPass}
	case "legacy":
		liveAuthModes = []identity.AuthMode{identity.AuthLegacy}
	}

	if *live {
		cred, err := identity.CompleteCredentialsFromEnv()
		if err != nil {
			t.Fatalf("Error setting up test suite: %s", err.Error())
		}
		registerOpenStackTests(cred, liveAuthModes)
	}
	registerLocalTests(allAuthModes)
	TestingT(t)
}
