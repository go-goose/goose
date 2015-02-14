package glance_test

import (
	"flag"
	"testing"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/client"
	"gopkg.in/goose.v1/glance"
	"gopkg.in/goose.v1/identity"
)

func Test(t *testing.T) { gc.TestingT(t) }

var live = flag.Bool("live", false, "Include live OpenStack (Canonistack) tests")

type GlanceSuite struct {
	glance *glance.Client
}

func (s *GlanceSuite) SetUpSuite(c *gc.C) {
	if !*live {
		c.Skip("-live not provided")
	}

	cred, err := identity.CompleteCredentialsFromEnv()
	c.Assert(err, gc.IsNil)
	client := client.NewClient(cred, identity.AuthUserPass, nil)
	c.Assert(err, gc.IsNil)
	s.glance = glance.New(client)
}

var suite = gc.Suite(&GlanceSuite{})

func (s *GlanceSuite) TestListImages(c *gc.C) {
	images, err := s.glance.ListImages()
	c.Assert(err, gc.IsNil)
	c.Assert(images, gc.Not(gc.HasLen), 0)
	for _, ir := range images {
		c.Assert(ir.Id, gc.Not(gc.Equals), "")
		c.Assert(ir.Name, gc.Not(gc.Equals), "")
		for _, l := range ir.Links {
			c.Assert(l.Href, gc.Matches, "https?://.*")
			c.Assert(l.Rel, gc.Matches, "self|bookmark|alternate")
		}
	}
}

func (s *GlanceSuite) TestListImagesDetail(c *gc.C) {
	images, err := s.glance.ListImagesDetail()
	c.Assert(err, gc.IsNil)
	c.Assert(images, gc.Not(gc.HasLen), 0)
	for _, ir := range images {
		c.Assert(ir.Created, gc.Matches, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*`)
		c.Assert(ir.Updated, gc.Matches, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*`)
		c.Assert(ir.Id, gc.Not(gc.Equals), "")
		c.Assert(ir.Status, gc.Not(gc.Equals), "")
		c.Assert(ir.Name, gc.Not(gc.Equals), "")
		for _, l := range ir.Links {
			c.Assert(l.Href, gc.Matches, "https?://.*")
			c.Assert(l.Rel, gc.Matches, "self|bookmark|alternate")
		}
		m := ir.Metadata
		c.Assert(m.Architecture, gc.Matches, "i386|x86_64|")
		c.Assert(m.State, gc.Matches, "active|available|")
	}
}

func (s *GlanceSuite) TestGetImageDetail(c *gc.C) {
	images, err := s.glance.ListImagesDetail()
	c.Assert(err, gc.IsNil)
	firstImage := images[0]
	ir, err := s.glance.GetImageDetail(firstImage.Id)
	c.Assert(err, gc.IsNil)
	c.Assert(ir.Created, gc.Matches, firstImage.Created)
	c.Assert(ir.Updated, gc.Matches, firstImage.Updated)
	c.Assert(ir.Name, gc.Equals, firstImage.Name)
	c.Assert(ir.Status, gc.Equals, firstImage.Status)
}
