package glance_test

import (
	"flag"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"launchpad.net/goose/glance"
	"launchpad.net/goose/identity"
	"reflect"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

var live = flag.Bool("live", false, "Include live OpenStack (Canonistack) tests")

type GlanceSuite struct {
	glance glance.GlanceClient
	// The id of an existing image which we will use in subsequent tests.
	imageId string
}

func (s *GlanceSuite) SetUpSuite(c *C) {
	if !*live {
		c.Skip("-live not provided")
	}

	cred := identity.CredentialsFromEnv()
	v := reflect.ValueOf(cred).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.String() == "" {
			c.Fatalf("required environment variable not set for credentials attribute: %s", t.Field(i).Name)
		}
	}
	client := client.NewOpenStackClient(cred, identity.AuthUserPass)
	err := client.Authenticate()
	if err != nil {
		c.Fatalf("OpenStack authentication failed for %s", cred.User)
	}
	c.Logf("client authenticated")
	s.glance = glance.NewGlanceClient(client)
	// For live testing, we'll use a known existing image.
	s.imageId = "ceee61e9-c7a5-4e51-ae61-6770e359e341"
}

var suite = Suite(&GlanceSuite{})

func (s *GlanceSuite) TestListImages(c *C) {
	images, err := s.glance.ListImages()
	c.Assert(err, IsNil)
	for _, ir := range images {
		c.Assert(ir.Id, Not(Equals), "")
		c.Assert(ir.Name, Not(Equals), "")
		for _, l := range ir.Links {
			c.Assert(l.Href, Matches, "https?://.*")
			c.Assert(l.Rel, Matches, "self|bookmark|alternate")
		}
	}
}

func (s *GlanceSuite) TestListImagesDetail(c *C) {
	images, err := s.glance.ListImagesDetail()
	c.Assert(err, IsNil)
	if len(images) < 1 {
		c.Fatalf("no images to list (expected at least 1)")
	}
	for _, ir := range images {
		c.Assert(ir.Created, Matches, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*`)
		c.Assert(ir.Updated, Matches, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*`)
		c.Assert(ir.Id, Not(Equals), "")
		c.Assert(ir.Status, Not(Equals), "")
		c.Assert(ir.Name, Not(Equals), "")
		for _, l := range ir.Links {
			c.Assert(l.Href, Matches, "https?://.*")
			c.Assert(l.Rel, Matches, "self|bookmark|alternate")
		}
		m := ir.Metadata
		c.Assert(m.Architecture, Matches, "i386|x86_64|")
		c.Assert(m.State, Matches, "active|available|")
	}
}

func (s *GlanceSuite) TestGetImageDetail(c *C) {
	ir, err := s.glance.GetImageDetail(s.imageId)
	c.Assert(err, IsNil)
	c.Assert(ir.Created, Matches, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*`)
	c.Assert(ir.Updated, Matches, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*`)
	c.Assert(ir.Name, Equals, "smoser-cloud-images-testing/ubuntu-precise-daily-amd64-server-20120519")
	c.Assert(ir.Status, Equals, "ACTIVE")
}
