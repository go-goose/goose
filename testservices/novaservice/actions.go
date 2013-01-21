package novaservice

import (
	"launchpad.net/goose/nova"
)

// GZ 2013-01-21: This should take map[int]interface{} but go disagrees
func generateId(existing_objects map[int]nova.SecurityGroup) int {
	for i := 1; ; i++ {
		_, ok := existing_objects[i]
		if !ok {
			return i
		}
	}
	panic("Could not generate a new id for Nova object")
}

func (n *Nova) MakeSecurityGroup(name, description string) {
	id := generateId(n.groups)
	n.groups[id] = nova.SecurityGroup{
		Id:          id,
		Name:        name,
		Description: description,
		TenantId:    n.tenantId,
	}
}
