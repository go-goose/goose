package glance_test

import (
	"flag"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"launchpad.net/goose/glance"
	"launchpad.net/goose/identity"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

var live = flag.Bool("live", false, "Include live OpenStack (Canonistack) tests")

type GlanceSuite struct {
	glance glance.Glance
}

func (s *GlanceSuite) SetUpSuite(c *C) {
	if !*live {
		c.Skip("-live not provided")
	}

	cred, err := identity.CompleteCredentialsFromEnv()
	if err != nil {
		c.Fatalf("Error setting up test suite: %s", err.Error())
	}
	client := client.NewOpenStackClient(cred, identity.AuthUserPass)
	err = client.Authenticate()
	if err != nil {
		c.Fatalf("OpenStack authentication failed for %s", cred.User)
	}
	c.Logf("client authenticated")
	s.glance = glance.NewClient(client)
}

var suite = Suite(&GlanceSuite{})

func (s *GlanceSuite) TestListImages(c *C) {
	images, err := s.glance.ListImages()
	c.Assert(err, IsNil)
	if len(images) < 1 {
		c.Fatalf("no images to list (expected at least 1)")
	}
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
	images, err := s.glance.ListImagesDetail()
	c.Assert(err, IsNil)
	firstImage := images[0]
	ir, err := s.glance.GetImageDetail(firstImage.Id)
	c.Assert(err, IsNil)
	c.Assert(ir.Created, Matches, firstImage.Created)
	c.Assert(ir.Updated, Matches, firstImage.Updated)
	c.Assert(ir.Name, Equals, firstImage.Name)
	c.Assert(ir.Status, Equals, firstImage.Status)
}
