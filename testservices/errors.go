package testservices

import "fmt"

var nameReference = map[int]string{
	400: "badRequest",
	401: "unauthorized",
	403: "forbidden",
	404: "itemNotFound",
	405: "badMethod",
	409: "conflictingRequest",
	413: "overLimit",
	415: "badMediaType",
	429: "overLimit",
	501: "notImplemented",
	503: "serviceUnavailable",
}

type ServerError struct {
	message string
	code    int
}

func (n *ServerError) Error() string {
	return fmt.Sprintf("%s: %s", n.Name(), n.message)
}

func (n *ServerError) Name() string {
	name, ok := nameReference[n.code]
	if !ok {
		return "computeFault"
	}
	return name
}

func NewNotFoundError(message string) *ServerError {
	return &ServerError{
		message: message,
		code:    404,
	}
}

func NewNoMoreFloatingIpsError() *ServerError {
	return &ServerError{
		message: "Zero floating ips available",
		code:    404,
	}
}

func NewIPLimitExceededError() *ServerError {
	return &ServerError{
		message: "Maximum number of floating ips exceeded",
		code:    413,
	}
}

func NewRateLimitExceededError() *ServerError {
	return &ServerError{
		message: "Retry limit exceeded",
		// XXX: hduran-8 I infered this from the python nova code, might be wrong
		code:    413,
	}
}

func NewAddFlavorError(id string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("A flavor with id %q already exists", id),
		code:    409,
	}
}

func NewNoSuchFlavorError(id string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("No such flavor %q", id),
		code:    404,
	}
}

func NewServerByIDNotFoundError(id string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("No such server %q", id),
		code:    404,
	}
}

func NewServerByNameNotFoundError(name string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("No such server named %q", name),
		code:    404,
	}
}

func NewServerAlreadyExistsError(id string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("A server with id %q already exists", id),
		code:    409,
	}
}

func NewSecurityGroupAlreadyExistsError(id string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("A security group with id %s already exists", id),
		code:    409,
	}
}

func NewSecurityGroupByIDNotFoundError(groupId string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("No such security group %s", groupId),
		code:    404,
	}
}

func NewSecurityGroupByNameNotFoundError(name string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("No such security group named %q", name),
		code:    404,
	}
}

func NewSecurityGroupRuleAlreadyExistsError(id string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("A security group rule with id %s already exists", id),
		code:    409,
	}
}

func NewCannotAddTwiceRuleToGroupError(ruleId, groupId string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("Cannot add twice rule %s to security group %s", ruleId, groupId),
		code:    409,
	}
}

func NewUnknownSecurityGroupError(groupId string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("Unknown source security group %s", groupId),
		code:    409,
	}
}

func NewSecurityGroupRuleNotFoundError(ruleId string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("No such security group rule %s", ruleId),
		code:    404,
	}
}

func NewServerBelongsToGroupError(serverId, groupId string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("Server %q already belongs to group %s", serverId, groupId),
		code:    409,
	}
}

func NewServerDoesNotBelongToGroupsError(serverId string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("Server %q does not belong to any groups", serverId),
		code:    400,
	}
}

func NewServerDoesNotBelongToGroupError(serverId, groupId string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("Server %q does not belong to group %s", serverId, groupId),
		code:    400,
	}
}

func NewFloatingIPExistsError(ipID string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("A floating IP with id %s already exists", ipID),
		code:    409,
	}
}

func NewFloatingIPNotFoundError(address string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("No such floating IP %q", address),
		code:    404,
	}
}

func NewServerHasFloatingIPError(serverId, ipId string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("Server %q already has floating IP %s", serverId, ipId),
		code:    409,
	}
}

func NewNoFloatingIPsToRemoveError(serverId string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("Server %q does not have any floating IPs to remove", serverId),
		code:    409,
	}
}

func NewNoFloatingIPsError(serverId, ipId string) *ServerError {
	return &ServerError{
		message: fmt.Sprintf("Server %q does not have floating IP %s", serverId, ipId),
		code:    404,
	}
}
