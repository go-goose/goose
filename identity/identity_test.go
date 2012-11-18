package identity

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/testing/envsuite"
	"os"
)

type CredentialsTestSuite struct {
	// Isolate all of these tests from the real Environ.
	envsuite.EnvSuite
}

var _ = Suite(&CredentialsTestSuite{})

func (s *CredentialsTestSuite) TestCredentialsFromEnv(c *C) {
	var scenarios = []struct {
		summary  string
		env      map[string]string
		username string
		password string
		tenant   string
		region   string
		authURL  string
	}{
		{summary: "Old 'NOVA' style creds",
			env: map[string]string{
				"NOVA_USERNAME":   "test-user",
				"NOVA_PASSWORD":   "test-pass",
				// TODO: JAM 20121118 There exists a 'tenant
				// name' and a 'tenant id'. Does
				// NOVA_PROJECT_ID map to the 'tenant id' or to
				// the tenant name? ~/.canonistack/novarc says
				// tenant_name.
				"NOVA_PROJECT_ID": "tenant-name",
				"NOVA_REGION":     "region",
			},
			username: "test-user",
			password: "test-pass",
			tenant:   "tenant-name",
			region:   "region",
		},
		{summary: "New 'OS' style environment",
			env: map[string]string{
				"OS_USERNAME":   "test-user",
				"OS_PASSWORD":   "test-pass",
				"OS_TENANT_NAME": "tenant-name",
				"OS_REGION_NAME":     "region",
			},
			username: "test-user",
			password: "test-pass",
			tenant:   "tenant-name",
			region:   "region",
		},
	}
	for _, scenario := range scenarios {
		for key, value := range scenario.env {
			os.Setenv(key, value)
		}
		creds := CredentialsFromEnv()
		c.Check(creds.URL, Equals, scenario.authURL)
		c.Check(creds.User, Equals, scenario.username)
		c.Check(creds.Secrets, Equals, scenario.password)
		c.Check(creds.Region, Equals, scenario.region)
		c.Check(creds.TenantName, Equals, scenario.tenant)
	}
}
