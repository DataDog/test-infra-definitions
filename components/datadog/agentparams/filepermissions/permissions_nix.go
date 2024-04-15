package filepermissions

import (
	"fmt"
	"strings"

	"github.com/DataDog/datadog-agent/pkg/util/optional"
	"github.com/DataDog/test-infra-definitions/common"
)

type UnixPermissionsOption = func(*UnixPermissions) error

// UnixPermissions represents the owner, group, and permissions of a file.
// Permissions are represented by a string (should be an octal). There is no check on the octal format.
type UnixPermissions struct {
	Owner       string
	Group       string
	Permissions string
}

var _ FilePermissions = (*UnixPermissions)(nil)

// NewUnixPermissions creates a new UnixPermissions object and applies the given options.
func NewUnixPermissions(options ...UnixPermissionsOption) optional.Option[FilePermissions] {
	p, err := common.ApplyOption(&UnixPermissions{}, options)

	if err != nil {
		return optional.NewNoneOption[FilePermissions]()
	}
	return optional.NewOption[FilePermissions](p)
}

// SetupPermissionsCommand returns a command that sets the owner, group, and permissions of a file.
func (p *UnixPermissions) SetupPermissionsCommand(path string) string {
	var commands []string

	if p.Owner != "" {
		commands = append(commands, fmt.Sprintf("chown %s %s", p.Owner, path))
	}

	if p.Group != "" {
		commands = append(commands, fmt.Sprintf("chgrp %s %s", p.Group, path))
	}

	if p.Permissions != "" {
		commands = append(commands, fmt.Sprintf("chmod %s %s", p.Permissions, path))
	}

	if len(commands) == 0 {
		return ""
	}
	return fmt.Sprintf(`sudo sh -c "%v"`, strings.Join(commands, " && "))
}

// ResetPermissionsCommand returns a command that resets the owner, group, and permissions of a file to default.
func (p *UnixPermissions) ResetPermissionsCommand(path string) string {
	return fmt.Sprintf("sudo chown ubuntu:ubuntu %s && sudo chmod 644 %s", path, path)
}

// WithOwner sets the owner of the file.
func WithOwner(owner string) UnixPermissionsOption {
	return func(p *UnixPermissions) error {
		p.Owner = owner
		return nil
	}
}

// WithGroup sets the group of the file.
func WithGroup(group string) UnixPermissionsOption {
	return func(p *UnixPermissions) error {
		p.Group = group
		return nil
	}
}

// WithPermissions sets the permissions of the file.
func WithPermissions(permissions string) UnixPermissionsOption {
	return func(p *UnixPermissions) error {
		p.Permissions = permissions
		return nil
	}
}
