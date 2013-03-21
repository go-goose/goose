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
	var id genericInstanceId
	if err := json.Unmarshal(b, &id); err != nil {
		return err
	}
	instId := id.String()
	if instId != "" {
		strId := instId
		jfip.InstanceId = &strId
	}
	*floatingIP = FloatingIP(jfip)
	return nil
}

func (floatingIP FloatingIP) MarshalJSON() ([]byte, error) {
	var jfip jsonFloatingIP = jsonFloatingIP(floatingIP)
	data, err := json.Marshal(&jfip)
	if err != nil {
		return nil, err
	}
	if floatingIP.InstanceId == nil {
		return data, nil
	}
	id := convertId(*floatingIP.InstanceId)
	return appendJSON(data, "instance_id", id)
}
