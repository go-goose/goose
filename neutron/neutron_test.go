package neutron_test

import (
	"flag"
	"testing"

	gc "gopkg.in/check.v1"

	"github.com/go-goose/goose/v4/identity"
)

var live = flag.Bool("live", false, "Include live OpenStack tests")

func Test(t *testing.T) {
	if *live {
		cred, err := identity.CompleteCredentialsFromEnv()
		if err != nil {
			t.Fatalf("Error setting up test suite: %s", err.Error())
		}
		registerOpenStackTests(cred)
	}
	registerLocalTests()
	gc.TestingT(t)
}
