package identityservice

import (
	"fmt"
	"strconv"
)

type Users struct {
	nextId int
	users  map[string]UserInfo
}

func (u *Users) AddUser(user, secret string) *UserInfo {
	u.nextId++
	// We may in future need to allow the tenant id to be specified, but for now it can be a fixed value.
	userInfo := &UserInfo{secret: secret, Id: strconv.Itoa(u.nextId), TenantId: "tenant"}
	u.users[user] = *userInfo
	userInfo, _ = u.authenticate(user, secret)
	return userInfo
}

func (u *Users) FindUser(token string) (*UserInfo, error) {
	for _, userInfo := range u.users {
		if userInfo.Token == token {
			return &userInfo, nil
		}
	}
	return nil, fmt.Errorf("No user with token %v exists", token)
}

const (
	notAuthorized = "The request you have made requires authentication."
	invalidUser   = "Invalid user / password"
)

func (u *Users) authenticate(username, password string) (*UserInfo, string) {
	userInfo, ok := u.users[username]
	if !ok {
		return nil, notAuthorized
	}
	if userInfo.secret != password {
		return nil, invalidUser
	}
	if userInfo.Token == "" {
		userInfo.Token = randomHexToken()
		u.users[username] = userInfo
	}
	return &userInfo, ""
}
