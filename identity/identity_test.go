package identity

import (
	"os"

	gc "gopkg.in/check.v1"

	goosehttp "gopkg.in/goose.v1/http"
	"gopkg.in/goose.v1/testing/envsuite"
)

type CredentialsTestSuite struct {
	// Isolate all of these tests from the real Environ.
	envsuite.EnvSuite
}

type NewAuthenticatorSuite struct{}

var _ = gc.Suite(&CredentialsTestSuite{})
var _ = gc.Suite(&NewAuthenticatorSuite{})

func (s *CredentialsTestSuite) TestCredentialsFromEnv(c *gc.C) {
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
				"NOVA_API_KEY":    "test-access-key",
				"EC2_SECRET_KEYS": "test-secret-key",
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
				"OS_USERNAME":    "test-user",
				"OS_PASSWORD":    "test-pass",
				"OS_ACCESS_KEY":  "test-access-key",
				"OS_SECRET_KEY":  "test-secret-key",
				"OS_TENANT_NAME": "tenant-name",
				"OS_REGION_NAME": "region",
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
		c.Check(creds.URL, gc.Equals, scenario.authURL)
		c.Check(creds.User, gc.Equals, scenario.username)
		c.Check(creds.Secrets, gc.Equals, scenario.password)
		c.Check(creds.Region, gc.Equals, scenario.region)
		c.Check(creds.TenantName, gc.Equals, scenario.tenant)
	}
}

func (s *CredentialsTestSuite) TestCompleteCredentialsFromEnvValid(c *gc.C) {
	env := map[string]string{
		"OS_AUTH_URL":    "http://auth",
		"OS_USERNAME":    "test-user",
		"OS_PASSWORD":    "test-pass",
		"OS_ACCESS_KEY":  "test-access-key",
		"OS_SECRET_KEY":  "test-secret-key",
		"OS_TENANT_NAME": "tenant-name",
		"OS_REGION_NAME": "region",
	}
	for key, value := range env {
		os.Setenv(key, value)
	}
	creds, err := CompleteCredentialsFromEnv()
	c.Assert(err, gc.IsNil)
	c.Check(creds.URL, gc.Equals, "http://auth")
	c.Check(creds.User, gc.Equals, "test-user")
	c.Check(creds.Secrets, gc.Equals, "test-pass")
	c.Check(creds.Region, gc.Equals, "region")
	c.Check(creds.TenantName, gc.Equals, "tenant-name")
}

// An error is returned if not all required environment variables are set.
func (s *CredentialsTestSuite) TestCompleteCredentialsFromEnvInvalid(c *gc.C) {
	env := map[string]string{
		"OS_AUTH_URL":    "http://auth",
		"OS_USERNAME":    "test-user",
		"OS_ACCESS_KEY":  "test-access-key",
		"OS_TENANT_NAME": "tenant-name",
		"OS_REGION_NAME": "region",
	}
	for key, value := range env {
		os.Setenv(key, value)
	}
	_, err := CompleteCredentialsFromEnv()
	c.Assert(err, gc.Not(gc.IsNil))
	c.Assert(err.Error(), gc.Matches, "required environment variable not set.*: Secrets")
}

func (s *CredentialsTestSuite) TestCompleteCredentialsFromEnvKeypair(c *gc.C) {
	env := map[string]string{
		"OS_AUTH_URL":    "http://auth",
		"OS_USERNAME":    "",
		"OS_PASSWORD":    "",
		"OS_ACCESS_KEY":  "test-access-key",
		"OS_SECRET_KEY":  "test-secret-key",
		"OS_TENANT_NAME": "tenant-name",
		"OS_REGION_NAME": "region",
	}
	for key, value := range env {
		os.Setenv(key, value)
	}
	creds, err := CompleteCredentialsFromEnv()
	c.Assert(err, gc.IsNil)
	c.Check(creds.URL, gc.Equals, "http://auth")
	c.Check(creds.User, gc.Equals, "test-access-key")
	c.Check(creds.Secrets, gc.Equals, "test-secret-key")
	c.Check(creds.Region, gc.Equals, "region")
	c.Check(creds.TenantName, gc.Equals, "tenant-name")
}

func (s *CredentialsTestSuite) TestCompleteCredentialsFromEnvKeypairCompatibleEnvVars(c *gc.C) {
	env := map[string]string{
		"OS_AUTH_URL":     "http://auth",
		"OS_USERNAME":     "",
		"OS_PASSWORD":     "",
		"NOVA_API_KEY":    "test-access-key",
		"EC2_SECRET_KEYS": "test-secret-key",
		"OS_TENANT_NAME":  "tenant-name",
		"OS_REGION_NAME":  "region",
	}
	for key, value := range env {
		os.Setenv(key, value)
	}
	creds, err := CompleteCredentialsFromEnv()
	c.Assert(err, gc.IsNil)
	c.Check(creds.URL, gc.Equals, "http://auth")
	c.Check(creds.User, gc.Equals, "test-access-key")
	c.Check(creds.Secrets, gc.Equals, "test-secret-key")
	c.Check(creds.Region, gc.Equals, "region")
	c.Check(creds.TenantName, gc.Equals, "tenant-name")
}

func (s *NewAuthenticatorSuite) TestUserPassNoHTTPClient(c *gc.C) {
	auth := NewAuthenticator(AuthUserPass, nil)
	userAuth, ok := auth.(*UserPass)
	c.Assert(ok, gc.Equals, true)
	c.Assert(userAuth.client, gc.NotNil)
}

func (s *NewAuthenticatorSuite) TestUserPassCustomHTTPClient(c *gc.C) {
	httpClient := goosehttp.New()
	auth := NewAuthenticator(AuthUserPass, httpClient)
	userAuth, ok := auth.(*UserPass)
	c.Assert(ok, gc.Equals, true)
	c.Assert(userAuth.client, gc.Equals, httpClient)
}

func (s *NewAuthenticatorSuite) TestKeyPairNoHTTPClient(c *gc.C) {
	auth := NewAuthenticator(AuthKeyPair, nil)
	keyPairAuth, ok := auth.(*KeyPair)
	c.Assert(ok, gc.Equals, true)
	c.Assert(keyPairAuth.client, gc.NotNil)
}

func (s *NewAuthenticatorSuite) TestKeyPairCustomHTTPClient(c *gc.C) {
	httpClient := goosehttp.New()
	auth := NewAuthenticator(AuthKeyPair, httpClient)
	keyPairAuth, ok := auth.(*KeyPair)
	c.Assert(ok, gc.Equals, true)
	c.Assert(keyPairAuth.client, gc.Equals, httpClient)
}

func (s *NewAuthenticatorSuite) TestLegacyNoHTTPClient(c *gc.C) {
	auth := NewAuthenticator(AuthLegacy, nil)
	legacyAuth, ok := auth.(*Legacy)
	c.Assert(ok, gc.Equals, true)
	c.Assert(legacyAuth.client, gc.NotNil)
}

func (s *NewAuthenticatorSuite) TestLegacyCustomHTTPClient(c *gc.C) {
	httpClient := goosehttp.New()
	auth := NewAuthenticator(AuthLegacy, httpClient)
	legacyAuth, ok := auth.(*Legacy)
	c.Assert(ok, gc.Equals, true)
	c.Assert(legacyAuth.client, gc.Equals, httpClient)
}

func (s *NewAuthenticatorSuite) TestUnknownMode(c *gc.C) {
	c.Assert(func() { NewAuthenticator(1235, nil) },
		gc.PanicMatches, "Invalid identity authorisation mode: 1235")
}
