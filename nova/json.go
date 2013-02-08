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

type JSONEntity struct {
	Entity `json:"-"`
}

func (entity *JSONEntity) UnmarshalJSON(b []byte) error {
	var e Entity
	if err := json.Unmarshal(b, &e); err != nil {
		return err
	}
	entity.Entity = e
	var id genericId
	if err := json.Unmarshal(b, &id); err != nil {
		return err
	}
	entity.Id = fmt.Sprint(id.Id)
	return nil
}

func (entity JSONEntity) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(&entity.Entity)
	if err != nil {
		return nil, err
	}
	id := convertId(entity.Entity.Id)
	data, err = appendJSON(data, "Id", id)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func convertEntities(je []JSONEntity) []Entity {
	entities := make([]Entity, len(je))
	for i, e := range je {
		entities[i] = e.Entity
	}
	return entities
}

type JSONFlavorDetail struct {
	FlavorDetail `json:"-"`
	genericId genericId `json:id`
}

func (flavorDetail *JSONFlavorDetail) UnmarshalJSON(b []byte) error {
	var fd FlavorDetail
	if err := json.Unmarshal(b, &fd); err != nil {
		return err
	}
	flavorDetail.FlavorDetail = fd
	var id genericId
	if err := json.Unmarshal(b, &id); err != nil {
		return err
	}
	flavorDetail.Id = fmt.Sprint(id.Id)
	return nil
}

func (flavorDetail JSONFlavorDetail) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(&flavorDetail.FlavorDetail)
	if err != nil {
		return nil, err
	}
	id := convertId(flavorDetail.FlavorDetail.Id)
	data, err = appendJSON(data, "Id", id)
	if err != nil {
		return nil, err
	}
	return data, nil
}

type JSONServerDetail struct {
	ServerDetail `json:"-"`
	genericId genericId `json:id`
}

type JSONServerDetailEntities struct {
	Image  JSONEntity   `json:"image"`
	Flavor JSONEntity   `json:"flavor"`
	Groups []JSONEntity `json:"security_groups"`
}

func (serverDetail *JSONServerDetail) UnmarshalJSON(b []byte) error {
	var sd ServerDetail
	if err := json.Unmarshal(b, &sd); err != nil {
		return err
	}
	serverDetail.ServerDetail = sd
	var id genericId
	if err := json.Unmarshal(b, &id); err != nil {
		return err
	}
	serverDetail.Id = fmt.Sprint(id.Id)
	var entities JSONServerDetailEntities
	if err := json.Unmarshal(b, &entities); err != nil {
		return err
	}
	serverDetail.Image = entities.Image.Entity
	serverDetail.Flavor = entities.Flavor.Entity
	serverDetail.Groups = convertEntities(entities.Groups)
	return nil
}

func (serverDetail JSONServerDetail) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(&serverDetail.ServerDetail)
	if err != nil {
		return nil, err
	}
	id := convertId(serverDetail.ServerDetail.Id)
	data, err = appendJSON(data, "Id", id)
	if err != nil {
		return nil, err
	}

	data, err = appendJSON(data, "image", JSONEntity{Entity: serverDetail.Image})
	if err != nil {
		return nil, err
	}

	data, err = appendJSON(data, "flavor", JSONEntity{Entity: serverDetail.Flavor})
	if err != nil {
		return nil, err
	}

	if serverDetail.Groups != nil {
		groups := make([]JSONEntity, len(serverDetail.Groups))
		for i, e := range serverDetail.Groups {
			groups[i] = JSONEntity{Entity: e}
		}
		data, err = appendJSON(data, "security_groups", groups)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

type JSONFloatingIP struct {
	FloatingIP `json:"-"`
}

type genericInstanceId struct {
	InstanceId interface{} `json:"instance_id"`
}

func (floatingIP *JSONFloatingIP) UnmarshalJSON(b []byte) error {
	var fip FloatingIP
	if err := json.Unmarshal(b, &fip); err != nil {
		return err
	}
	floatingIP.FloatingIP = fip
	var id genericInstanceId
	if err := json.Unmarshal(b, &id); err != nil {
		return err
	}
	if id.InstanceId != nil && id.InstanceId != "" {
		strId := fmt.Sprint(id.InstanceId)
		floatingIP.InstanceId = &strId
	}
	return nil
}

func (floatingIP JSONFloatingIP) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(&floatingIP.FloatingIP)
	if err != nil {
		return nil, err
	}
	var id interface{}
	if floatingIP.FloatingIP.InstanceId == nil {
		return data, nil
	}
	id = convertId(*floatingIP.FloatingIP.InstanceId)
	data, err = appendJSON(data, "instance_id", id)
	if err != nil {
		return nil, err
	}
	return data, nil
}
