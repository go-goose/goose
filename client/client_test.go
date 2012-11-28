package client_test

import (
	"flag"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/identity"
	"testing"
)

var live = flag.Bool("live", false, "Include live OpenStack (Canonistack) tests")
var authMethodName = flag.String("auth_method", "userpass", "The authentication mode to use [legacy|userpass]")

func Test(t *testing.T) {
	cred, err := identity.CompleteCredentialsFromEnv()
	if err != nil {
		t.Fatalf("Error setting up test suite: %s", err.Error())
	}
	var authMethod identity.AuthMethod
	switch *authMethodName {
	default:
		t.Fatalf("Invalid auth method specified: %s", *authMethodName)
	case "":
	case "userpass":
		authMethod = identity.AuthUserPass
	case "legacy":
		authMethod = identity.AuthLegacy
	}

	if *live {
		registerOpenStackTests(cred, authMethod)
	}
	registerLocalTests(cred, authMethod)
	TestingT(t)
}
