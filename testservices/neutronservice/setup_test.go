package neutronservice

import (
	"testing"

	gc "gopkg.in/check.v1"

	"github.com/go-goose/goose/v5/neutron"
)

func Test(t *testing.T) {
	gc.TestingT(t)
}

// checkGroupsInList checks that every group in groups is in groupList.
func checkGroupsInList(c *gc.C, groups, groupList []neutron.SecurityGroupV2) {
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

// checkPortsInList checks that every port in ports is in portList.
func checkPortsInList(c *gc.C, ports, portList []neutron.PortV2) {
	for _, g := range ports {
		for _, gr := range portList {
			if g.Id == gr.Id {
				c.Assert(g, gc.DeepEquals, gr)
				return
			}
		}
		c.Fail()
	}
}
