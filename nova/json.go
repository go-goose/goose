// JSON marshaling and unmarshaling support for Openstack compute data structures.
// This package deals with the difference between the API and on-the-wire data types.
// Differences include encoding entity IDs as string vs int, depending on the Openstack
// variant used.
//
// The marshaling support is included primarily for use by the test doubles. It needs to be
// included here since Go requires methods to implemented in the same package as their receiver.

package nova

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type genericId struct {
	Id interface{} `json:"id"`
}

func (id genericId) String() string {
	if id.Id == nil {
		return ""
	}
	if fid, ok := id.Id.(float64); ok {
		return fmt.Sprint(int(fid))
	}
	return fmt.Sprint(id.Id)
}

var useNumericIds bool = false

// convertId returns the id as either a string or an int depending on what
// implementation of Openstack we are emulating.
func convertId(id string) interface{} {
	if !useNumericIds {
		return id
	}
	result, err := strconv.Atoi(id)
	if err != nil {
		panic(err)
	}
	return result
}

// appendJSON marshals the given attribute value and appends it as an encoded value to the given json data.
// The newly encode (attr, value) is inserted just before the closing "}" in the json data.
func appendJSON(data []byte, attr string, value interface{}) ([]byte, error) {
	newData, err := json.Marshal(&value)
	if err != nil {
		return nil, err
	}
	strData := string(data)
	result := fmt.Sprintf(`%s, "%s":%s}`, strData[:len(strData)-1], attr, string(newData))
	return []byte(result), nil
}

type jsonEntity Entity

func (entity *Entity) UnmarshalJSON(b []byte) error {
	var je jsonEntity = jsonEntity(*entity)
	if err := json.Unmarshal(b, &je); err != nil {
		return err
	}
	var id genericId
	if err := json.Unmarshal(b, &id); err != nil {
		return err
	}
	je.Id = id.String()
	*entity = Entity(je)
	return nil
}

func (entity Entity) MarshalJSON() ([]byte, error) {
	var je jsonEntity = jsonEntity(entity)
	data, err := json.Marshal(&je)
	if err != nil {
		return nil, err
	}
	id := convertId(entity.Id)
	return appendJSON(data, "Id", id)
}

type jsonFlavorDetail FlavorDetail

func (flavorDetail *FlavorDetail) UnmarshalJSON(b []byte) error {
	var jfd jsonFlavorDetail = jsonFlavorDetail(*flavorDetail)
	if err := json.Unmarshal(b, &jfd); err != nil {
		return err
	}
	var id genericId
	if err := json.Unmarshal(b, &id); err != nil {
		return err
	}
	jfd.Id = id.String()
	*flavorDetail = FlavorDetail(jfd)
	return nil
}

func (flavorDetail FlavorDetail) MarshalJSON() ([]byte, error) {
	var jfd jsonFlavorDetail = jsonFlavorDetail(flavorDetail)
	data, err := json.Marshal(&jfd)
	if err != nil {
		return nil, err
	}
	id := convertId(flavorDetail.Id)
	return appendJSON(data, "Id", id)
}

type jsonServerDetail ServerDetail

func (serverDetail *ServerDetail) UnmarshalJSON(b []byte) error {
	var jsd jsonServerDetail = jsonServerDetail(*serverDetail)
	if err := json.Unmarshal(b, &jsd); err != nil {
		return err
	}
	var id genericId
	if err := json.Unmarshal(b, &id); err != nil {
		return err
	}
	jsd.Id = id.String()
	*serverDetail = ServerDetail(jsd)
	return nil
}

func (serverDetail ServerDetail) MarshalJSON() ([]byte, error) {
	var jsd jsonServerDetail = jsonServerDetail(serverDetail)
	data, err := json.Marshal(&jsd)
	if err != nil {
		return nil, err
	}
	id := convertId(serverDetail.Id)
	return appendJSON(data, "Id", id)
}

type genericInstanceId struct {
	InstanceId interface{} `json:"instance_id"`
}

func (id genericInstanceId) String() string {
	if id.InstanceId == nil {
		return ""
	}
	if fid, ok := id.InstanceId.(float64); ok {
		return fmt.Sprint(int(fid))
	}
	return fmt.Sprint(id.InstanceId)
}

type jsonFloatingIP FloatingIP

func (floatingIP *FloatingIP) UnmarshalJSON(b []byte) error {
	var jfip jsonFloatingIP = jsonFloatingIP(*floatingIP)
	if err := json.Unmarshal(b, &jfip); err != nil {
		return err
	}
	var instId genericInstanceId
	if err := json.Unmarshal(b, &instId); err != nil {
		return err
	}
	instIdStr := instId.String()
	if instIdStr != "" {
		strId := instIdStr
		jfip.InstanceId = &strId
	}
	var id genericId
	if err := json.Unmarshal(b, &id); err != nil {
		return err
	}
	jfip.Id = id.String()
	*floatingIP = FloatingIP(jfip)
	return nil
}

func (floatingIP FloatingIP) MarshalJSON() ([]byte, error) {
	var jfip jsonFloatingIP = jsonFloatingIP(floatingIP)
	data, err := json.Marshal(&jfip)
	if err != nil {
		return nil, err
	}
	id := convertId(floatingIP.Id)
	data, err = appendJSON(data, "Id", id)
	if err != nil {
		return nil, err
	}
	if floatingIP.InstanceId == nil {
		return data, nil
	}
	instId := convertId(*floatingIP.InstanceId)
	return appendJSON(data, "instance_id", instId)
}

type jsonSecurityGroup SecurityGroup

func (securityGroup *SecurityGroup) UnmarshalJSON(b []byte) error {
	var jsg jsonSecurityGroup = jsonSecurityGroup(*securityGroup)
	if err := json.Unmarshal(b, &jsg); err != nil {
		return err
	}
	var id genericId
	if err := json.Unmarshal(b, &id); err != nil {
		return err
	}
	jsg.Id = id.String()
	*securityGroup = SecurityGroup(jsg)
	return nil
}

func (securityGroup SecurityGroup) MarshalJSON() ([]byte, error) {
	var jsg jsonSecurityGroup = jsonSecurityGroup(securityGroup)
	data, err := json.Marshal(&jsg)
	if err != nil {
		return nil, err
	}
	id := convertId(securityGroup.Id)
	return appendJSON(data, "Id", id)
}

type genericParentGroupId struct {
	ParentGroupId interface{} `json:"parent_group_id"`
}

func (id genericParentGroupId) String() string {
	if id.ParentGroupId == nil {
		return ""
	}
	if pgid, ok := id.ParentGroupId.(float64); ok {
		return fmt.Sprint(int(pgid))
	}
	return fmt.Sprint(id.ParentGroupId)
}

type jsonSecurityGroupRule SecurityGroupRule

func (securityGroupRule *SecurityGroupRule) UnmarshalJSON(b []byte) error {
	var jsgr jsonSecurityGroupRule = jsonSecurityGroupRule(*securityGroupRule)
	if err := json.Unmarshal(b, &jsgr); err != nil {
		return err
	}
	var id genericId
	if err := json.Unmarshal(b, &id); err != nil {
		return err
	}
	jsgr.Id = id.String()
	var pgid genericParentGroupId
	if err := json.Unmarshal(b, &pgid); err != nil {
		return err
	}
	jsgr.ParentGroupId = pgid.String()
	*securityGroupRule = SecurityGroupRule(jsgr)
	return nil
}

func (securityGroupRule SecurityGroupRule) MarshalJSON() ([]byte, error) {
	var jsgr jsonSecurityGroupRule = jsonSecurityGroupRule(securityGroupRule)
	data, err := json.Marshal(&jsgr)
	if err != nil {
		return nil, err
	}
	id := convertId(securityGroupRule.Id)
	data, err = appendJSON(data, "Id", id)
	if err != nil {
		return nil, err
	}
	if securityGroupRule.ParentGroupId == "" {
		return data, nil
	}
	id = convertId(securityGroupRule.ParentGroupId)
	return appendJSON(data, "parent_group_id", id)
}

type genericGroupId struct {
	GroupId interface{} `json:"group_id"`
}

func (id genericGroupId) String() string {
	if id.GroupId == nil {
		return ""
	}
	if gid, ok := id.GroupId.(float64); ok {
		return fmt.Sprint(int(gid))
	}
	return fmt.Sprint(id.GroupId)
}

type jsonRuleInfo RuleInfo

func (ruleInfo *RuleInfo) UnmarshalJSON(b []byte) error {
	var jri jsonRuleInfo = jsonRuleInfo(*ruleInfo)
	if err := json.Unmarshal(b, &jri); err != nil {
		return err
	}

	var pgid genericParentGroupId
	if err := json.Unmarshal(b, &pgid); err != nil {
		return err
	}
	jri.ParentGroupId = pgid.String()

	var gid genericGroupId
	if err := json.Unmarshal(b, &gid); err != nil {
		return err
	}
	groupId := gid.String()
	if groupId != "" {
		strId := groupId
		jri.GroupId = &strId
	}
	*ruleInfo = RuleInfo(jri)
	return nil
}

func (ruleInfo RuleInfo) MarshalJSON() ([]byte, error) {
	var jri jsonRuleInfo = jsonRuleInfo(ruleInfo)
	data, err := json.Marshal(&jri)
	if err != nil {
		return nil, err
	}
	id := convertId(ruleInfo.ParentGroupId)
	data, err = appendJSON(data, "parent_group_id", id)
	if err != nil {
		return nil, err
	}
	if ruleInfo.GroupId == nil {
		return data, nil
	}
	id = convertId(*ruleInfo.GroupId)
	return appendJSON(data, "group_id", id)
}
