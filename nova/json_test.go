package nova_test

import (
	"encoding/json"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/nova"
)

type JsonSuite struct {
}

var _ = gc.Suite(&JsonSuite{})

func (s *JsonSuite) SetUpSuite(c *gc.C) {
	nova.UseNumericIds(true)
}

func (s *JsonSuite) assertMarshallRoundtrip(c *gc.C, value interface{}, unmarshalled interface{}) {
	data, err := json.Marshal(value)
	c.Assert(err, gc.IsNil)
	err = json.Unmarshal(data, &unmarshalled)
	c.Assert(err, gc.IsNil)
	c.Assert(unmarshalled, gc.DeepEquals, value)
}

// The following tests all check that unmarshalling of Ids with values > 1E6
// works properly.

func (s *JsonSuite) TestMarshallEntityLargeIntId(c *gc.C) {
	entity := nova.Entity{Id: "2000000", Name: "test"}
	var unmarshalled nova.Entity
	s.assertMarshallRoundtrip(c, &entity, &unmarshalled)
}

func (s *JsonSuite) TestMarshallFlavorDetailLargeIntId(c *gc.C) {
	fd := nova.FlavorDetail{Id: "2000000", Name: "test"}
	var unmarshalled nova.FlavorDetail
	s.assertMarshallRoundtrip(c, &fd, &unmarshalled)
}

func (s *JsonSuite) TestMarshallServerDetailLargeIntId(c *gc.C) {
	fd := nova.Entity{Id: "2000000", Name: "test"}
	im := nova.Entity{Id: "2000000", Name: "test"}
	sd := nova.ServerDetail{Id: "2000000", Name: "test", Flavor: fd, Image: im}
	var unmarshalled nova.ServerDetail
	s.assertMarshallRoundtrip(c, &sd, &unmarshalled)
}

func (s *JsonSuite) TestMarshallFloatingIPLargeIntId(c *gc.C) {
	id := "3000000"
	fip := nova.FloatingIP{Id: "2000000", InstanceId: &id}
	var unmarshalled nova.FloatingIP
	s.assertMarshallRoundtrip(c, &fip, &unmarshalled)
}

func (s *JsonSuite) TestUnmarshallFloatingIPNilStrings(c *gc.C) {
	var fip nova.FloatingIP
	data := []byte(`{"instance_id": null, "ip": "10.1.1.1", "fixed_ip": null, "id": "12345", "pool": "Ext-Net"}`)
	err := json.Unmarshal(data, &fip)
	c.Assert(err, gc.IsNil)
	expected := nova.FloatingIP{
		Id:         "12345",
		IP:         "10.1.1.1",
		Pool:       "Ext-Net",
		FixedIP:    nil,
		InstanceId: nil,
	}
	c.Assert(fip, gc.DeepEquals, expected)
}

func (s *JsonSuite) TestUnmarshallRuleInfoNilStrings(c *gc.C) {
	var ri nova.RuleInfo
	data := []byte(`{"group_id": null, "parent_group_id": "12345"}`)
	err := json.Unmarshal(data, &ri)
	c.Assert(err, gc.IsNil)
	expected := nova.RuleInfo{
		GroupId:       nil,
		ParentGroupId: "12345",
	}
	c.Assert(ri, gc.DeepEquals, expected)
}
