package envsuite

import (
	"os"
	"testing"

	gc "gopkg.in/check.v1"
)

type EnvTestSuite struct {
	EnvSuite
}

func Test(t *testing.T) {
	gc.TestingT(t)
}

var _ = gc.Suite(&EnvTestSuite{})

func (s *EnvTestSuite) TestGrabsCurrentEnvironment(c *gc.C) {
	envsuite := &EnvSuite{}
	// EnvTestSuite is an EnvSuite, so we should have already isolated
	// ourselves from the world. So we set a single env value, and we
	// assert that SetUpSuite is able to see that.
	os.Setenv("TEST_KEY", "test-value")
	envsuite.SetUpSuite(c)
	c.Assert(envsuite.environ, gc.DeepEquals, []string{"TEST_KEY=test-value"})
}

func (s *EnvTestSuite) TestClearsEnvironment(c *gc.C) {
	envsuite := &EnvSuite{}
	os.Setenv("TEST_KEY", "test-value")
	envsuite.SetUpSuite(c)
	// SetUpTest should reset the current environment back to being
	// completely empty.
	envsuite.SetUpTest(c)
	c.Assert(os.Getenv("TEST_KEY"), gc.Equals, "")
	c.Assert(os.Environ(), gc.DeepEquals, []string{})
}

func (s *EnvTestSuite) TestRestoresEnvironment(c *gc.C) {
	envsuite := &EnvSuite{}
	os.Setenv("TEST_KEY", "test-value")
	envsuite.SetUpSuite(c)
	envsuite.SetUpTest(c)
	envsuite.TearDownTest(c)
	c.Assert(os.Getenv("TEST_KEY"), gc.Equals, "test-value")
	c.Assert(os.Environ(), gc.DeepEquals, []string{"TEST_KEY=test-value"})
}
