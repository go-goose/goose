package nova_test

import (
	"flag"
	"reflect"
	"strings"
	"testing"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v3/identity"
)

// instanceDetails specify parameters used to start a test machine for the live tests.
type instanceDetails struct {
	useNeutronNetworking bool
	flavor               string
	imageId              string
	network              string
	vendor               string
}

// Out-of-the-box, we support live testing using Canonistack.
var testConstraints = map[string]instanceDetails{
	"canonistack": {
		flavor: "m1.tiny", imageId: "f2ca48ce-30d5-4f1f-9075-12e64510368d"},
}

var live = flag.Bool("live", false, "Include live OpenStack tests")
var netType = flag.String("netType", "neutron", "Use Neutron or Nova Networking? Default is Neutron.")
var vendor = flag.String("vendor", "", "The Openstack vendor to test against")
var imageId = flag.String("image", "", "The image id for which a test service is to be started")
var flavor = flag.String("flavor", "", "The flavor of the test service")
var network = flag.String("network", "", "The uuid private network of the test service")

func Test(t *testing.T) {
	if *live {
		// We can either specify a vendor, or imageId and flavor separately.
		var testDetails instanceDetails
		if *vendor != "" {
			var ok bool
			if testDetails, ok = testConstraints[*vendor]; !ok {
				keys := reflect.ValueOf(testConstraints).MapKeys()
				t.Fatalf("Unknown vendor %s. Must be one of %s", *vendor, keys)
			}
			testDetails.vendor = *vendor
		} else {
			if *imageId == "" {
				t.Fatalf("Must specify image id to use for test instance, "+
					"eg %s for Canonistack", "-image c876e5fe-abb0-41f0-8f29-f0b47481f523")
			}
			if *flavor == "" {
				t.Fatalf("Must specify flavor to use for test instance, "+
					"eg %s for Canonistack", "-flavor m1.tiny")
			}
			if *network == "" {
				t.Fatalf("Must specify network by UUID to use for test instance, "+
					"eg %s for Canonistack", "-network 4e408154-e71a-4702-a8fd-edb8df1b4e6c")
			}
			var useNeutronNetworking bool
			if strings.ToLower(*netType) == "neutron" {
				useNeutronNetworking = true
			} else if strings.ToLower(*netType) == "nova" {
				useNeutronNetworking = false
			} else {
				t.Fatalf("netType must be neutron or nova")
			}
			testDetails = instanceDetails{useNeutronNetworking, *flavor, *imageId, *network, ""}
		}
		cred, err := identity.CompleteCredentialsFromEnv()
		if err != nil {
			t.Fatalf("Error setting up test suite: %s", err.Error())
		}
		registerOpenStackTests(cred, testDetails)
	}
	registerLocalTests()
	gc.TestingT(t)
}
