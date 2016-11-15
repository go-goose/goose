package neutronservice

import (
	"testing"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/neutron"
)

func Test(t *testing.T) {
	gc.TestingT(t)
}

// checkGroupsInList checks that every group in groups is in groupList.
func checkGroupsInList(c *gc.C, groups []neutron.SecurityGroupV2, groupList []neutron.SecurityGroupV2) {
	for _, g := range groups {
		for _, gr := range groupList {
			if g.Id == gr.Id {
				c.Assert(g, gc.DeepEquals, gr)
				return
			}
		}
		c.Fail()
	}
}
