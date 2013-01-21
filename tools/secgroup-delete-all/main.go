package main

import (
	"fmt"
	"io"
	"launchpad.net/gnuflag"
	"launchpad.net/goose/client"
	"launchpad.net/goose/identity"
	"launchpad.net/goose/nova"
	"os"
)

// DeleteAll destroys all security groups except the default
func DeleteAll(w io.Writer, authMode identity.AuthMethod) (err error) {
	creds, err := identity.CompleteCredentialsFromEnv()
	if err != nil {
		return err
	}
	osc := client.NewClient(creds, authMode, nil)
	osn := nova.New(osc)
	groups, err := osn.ListSecurityGroups()
	if err != nil {
		return err
	}
	deleted := 0
	failed := 0
	for _, group := range groups {
		if group.Name != "default" {
			err := osn.DeleteSecurityGroup(group.Id)
			if err != nil {
				return err
				failed += 1
			} else {
				deleted += 1
			}
		}
	}
	if deleted != 0 {
		fmt.Fprintf(w, "%d security groups deleted.\n", deleted)
	} else if failed == 0 {
		fmt.Fprint(w, "No security groups to delete.\n")
	}
	if failed != 0 {
		fmt.Fprintf(w, "%d security groups could not be deleted.\n", failed)
	}
	return nil
}

var authModeFlag = gnuflag.String("auth-mode", "userpass", "type of authentication to use")

var authModes = map[string]identity.AuthMethod{
	"userpass": identity.AuthUserPass,
	"legacy":   identity.AuthLegacy,
}

func main() {
	gnuflag.Parse(true)
	mode, ok := authModes[*authModeFlag]
	if !ok {
		fmt.Fprintf(os.Stderr, "error: no such auth-mode %q\n", *authModeFlag)
		os.Exit(1)
	}
	err := DeleteAll(os.Stdout, mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
