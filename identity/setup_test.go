package identity_test

import (
	"flag"
	"testing"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v2/identity"
)

var live = flag.Bool("live", false, "Include live OpenStack (Canonistack) tests")
var v3 = flag.Bool("v3", false, "Run keystone v3 tests instead of v2 (requires live flag)")

func Test(t *testing.T) {
	if *live {
		cred, err := identity.CompleteCredentialsFromEnv()
		if err != nil {
			t.Fatalf("Error setting up test suite: %s", err.Error())
		}
		if *v3 {
			registerOpenStackTests(cred, identity.AuthUserPassV3)
		} else {
			registerOpenStackTests(cred, identity.AuthUserPass)
		}
	}
	registerLocalTests(identity.AuthUserPassV3)
	registerLocalTests(identity.AuthUserPass)
	gc.TestingT(t)
}
