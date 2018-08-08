package identity

import (
	"os"

	gc "gopkg.in/check.v1"

	goosehttp "gopkg.in/goose.v2/http"
	"gopkg.in/goose.v2/testing/envsuite"
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
		domain   string
		authURL  string
	}{
		{summary: "Old 'NOVA' style creds",
			env: map[string]string{
				"OS_AUTH_URL":     "http://auth/v2",
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
			authURL:  "http://auth/v2",
		},
		{summary: "New 'OS' style environment",
			env: map[string]string{
				"OS_AUTH_URL":    "http://auth/v2",
				"OS_USERNAME":    "test-user",
				"OS_PASSWORD":    "test-pass",
				"OS_ACCESS_KEY":  "test-access-key",
				"OS_SECRET_KEY":  "test-secret-key",
				"OS_TENANT_NAME": "tenant-name",
				"OS_REGION_NAME": "region",
				"OS_DOMAIN_NAME": "domain-name",
			},
			username: "test-user",
			password: "test-pass",
			tenant:   "tenant-name",
			region:   "region",
			domain:   "domain-name",
			authURL:  "http://auth/v2",
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
		"OS_AUTH_URL":            "http://auth",
		"OS_USERNAME":            "test-user",
		"OS_PASSWORD":            "test-pass",
		"OS_ACCESS_KEY":          "test-access-key",
		"OS_SECRET_KEY":          "test-secret-key",
		"OS_PROJECT_NAME":        "tenant-name",
		"OS_REGION_NAME":         "region",
		"OS_DOMAIN_NAME":         "domain-name",
		"OS_PROJECT_DOMAIN_NAME": "project-domain-name",
		"OS_USER_DOMAIN_NAME":    "user-domain-name",
		// ignored because user and project domains set
		"OS_DEFAULT_DOMAIN_NAME": "default-domain-name",
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
	c.Check(creds.Domain, gc.Equals, "domain-name")
	c.Check(creds.ProjectDomain, gc.Equals, "project-domain-name")
	c.Check(creds.UserDomain, gc.Equals, "user-domain-name")
}

func (s *CredentialsTestSuite) TestCompleteCredentialsFromEnvDefaultDomain(c *gc.C) {
	env := map[string]string{
		"OS_AUTH_URL":            "http://auth",
		"OS_USERNAME":            "test-user",
		"OS_PASSWORD":            "test-pass",
		"OS_ACCESS_KEY":          "test-access-key",
		"OS_SECRET_KEY":          "test-secret-key",
		"OS_PROJECT_NAME":        "tenant-name",
		"OS_REGION_NAME":         "region",
		"OS_DOMAIN_NAME":         "domain-name",
		"OS_DEFAULT_DOMAIN_NAME": "default-domain-name",
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
	c.Check(creds.Domain, gc.Equals, "domain-name")
	c.Check(creds.ProjectDomain, gc.Equals, "default-domain-name")
	c.Check(creds.UserDomain, gc.Equals, "default-domain-name")
}

func (s *CredentialsTestSuite) TestCompleteCredentialsFromEnvVersion(c *gc.C) {
	env := map[string]string{
		"OS_AUTH_URL":            "http://auth",
		"OS_USERNAME":            "test-user",
		"OS_PASSWORD":            "test-pass",
		"OS_ACCESS_KEY":          "test-access-key",
		"OS_SECRET_KEY":          "test-secret-key",
		"OS_PROJECT_NAME":        "tenant-name",
		"OS_REGION_NAME":         "region",
		"OS_DOMAIN_NAME":         "domain-name",
		"OS_AUTH_VERSION":        "v3",
		"OS_DEFAULT_DOMAIN_NAME": "default-domain-name",
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
	c.Check(creds.Domain, gc.Equals, "domain-name")
	c.Check(creds.ProjectDomain, gc.Equals, "default-domain-name")
	c.Check(creds.UserDomain, gc.Equals, "default-domain-name")
	c.Check(creds.Version, gc.Equals, "v3")
}

func (s *CredentialsTestSuite) TestCompleteCredentialsFromEnvProjectID(c *gc.C) {
	env := map[string]string{
		"OS_AUTH_URL":             "http://auth",
		"OS_USERNAME":             "test-user",
		"OS_PASSWORD":             "test-pass",
		"OS_ACCESS_KEY":           "test-access-key",
		"OS_SECRET_KEY":           "test-secret-key",
		"OS_PROJECT_ID":           "tenant-id",
		"OS_REGION_NAME":          "region",
		"OS_DOMAIN_NAME":          "domain-name",
		"OS_IDENTITY_API_VERSION": "v3",
		"OS_DEFAULT_DOMAIN_NAME":  "default-domain-name",
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
	c.Check(creds.Domain, gc.Equals, "domain-name")
	c.Check(creds.ProjectDomain, gc.Equals, "default-domain-name")
	c.Check(creds.UserDomain, gc.Equals, "default-domain-name")
	c.Check(creds.Version, gc.Equals, "v3")
	c.Check(creds.TenantID, gc.Equals, "tenant-id")
	c.Check(creds.TenantName, gc.Equals, "")
}

// An error is returned if not all required environment variables are set.
func (s *CredentialsTestSuite) TestCompleteCredentialsFromEnvInvalid(c *gc.C) {
	env := map[string]string{
		"OS_AUTH_URL":     "http://auth",
		"OS_USERNAME":     "test-user",
		"OS_ACCESS_KEY":   "test-access-key",
		"OS_PROJECT_NAME": "tenant-name",
		"OS_REGION_NAME":  "region",
		"OS_DOMAIN_NAME":  "domain-name",
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
		"OS_AUTH_URL":     "http://auth",
		"OS_USERNAME":     "",
		"OS_PASSWORD":     "",
		"OS_ACCESS_KEY":   "test-access-key",
		"OS_SECRET_KEY":   "test-secret-key",
		"OS_PROJECT_NAME": "tenant-name",
		"OS_REGION_NAME":  "region",
		"OS_DOMAIN_NAME":  "domain-name",
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
	c.Check(creds.Domain, gc.Equals, "domain-name")
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
		"OS_DOMAIN_NAME":  "domain-name",
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
	c.Check(creds.Domain, gc.Equals, "domain-name")
}

func (s *CredentialsTestSuite) TestCompleteCredentialsCheckProjectNameAliasVars(c *gc.C) {
	env := map[string]string{
		// required env vars
		"OS_AUTH_URL":     "http://auth",
		"NOVA_API_KEY":    "test-access-key",
		"EC2_SECRET_KEYS": "test-secret-key",
		"OS_REGION_NAME":  "region",

		// project Name Aliases
		"OS_PROJECT_NAME": "project-name",
		"OS_TENANT_NAME":  "tenant-name",
		"NOVA_PROJECT_ID": "nova-project-id",
		"OS_PROJECT_ID":   "project-id",
		"OS_TENANT_ID":    "tenant-id",
	}

	for key, value := range env {
		os.Setenv(key, value)
	}

	// Environment variables aliases are checked and set in the order
	// defined by their `Cred` slices. The first one found to be set and to
	// a value other than empty string is used.
	for _, key := range CredEnvTenantName {
		creds, err := CompleteCredentialsFromEnv()
		c.Assert(err, gc.IsNil)
		c.Check(creds.TenantName, gc.Equals, env[key])
		os.Unsetenv(key)
	}

	// same for TenantID
	for _, key := range CredEnvTenantID {
		creds, err := CompleteCredentialsFromEnv()
		c.Assert(err, gc.IsNil)
		c.Check(creds.TenantID, gc.Equals, env[key])
		os.Unsetenv(key)
	}
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
