package filepermissions

import (
	"fmt"

	"github.com/DataDog/datadog-agent/pkg/util/optional"
	"github.com/DataDog/test-infra-definitions/common"
)

type WindowsPermissionsOption = func(*WindowsPermissions) error

// WindowsPermissions contains the information to configure the permissions of a file on Windows.
type WindowsPermissions struct {
	// RemoveDefaultPermissions removes the default permissions from the file. This is useful when you want to set permissions for secrets management.
	RemoveDefaultPermissions bool

	// If you are familiar with the icacls command, you can provide a custom command directly.
	IcaclsCommand string
}

var _ FilePermissions = (*WindowsPermissions)(nil)

// NewWindowsPermissions creates a new WindowsPermissions object and applies the given options.
func NewWindowsPermissions(options ...WindowsPermissionsOption) optional.Option[FilePermissions] {
	p, err := common.ApplyOption(&WindowsPermissions{}, options)

	if err != nil {
		panic("Could not create WindowsPermissions: " + err.Error())
	}
	return optional.NewOption[FilePermissions](p)
}

// SetupPermissionsCommand returns a command that sets the permissions of a file. It relies on the icacls command.
func (p *WindowsPermissions) SetupPermissionsCommand(path string) string {
	cmd := ""
	if p.RemoveDefaultPermissions {
		cmd = fmt.Sprintf(`icacls "%v" /remove (Get-LocalUser) "Everyone" "Users" "Administrators" "SYSTEM" "Authenticated Users" "CREATOR OWNER" /inheritance:r /t /c /l;`, path)
	}
	if p.IcaclsCommand != "" {
		return fmt.Sprintf(`%v icacls "%v" %v`, cmd, path, p.IcaclsCommand)
	}
	return cmd
}

// ResetPermissionsCommand returns a command that resets the owner, group, and permissions of a file to default.
func (p *WindowsPermissions) ResetPermissionsCommand(path string) string {
	return fmt.Sprintf("icacls “%v” /reset /t /c /l", path)
}

// WithIcaclsCommand sets the icacls command to use.
func WithIcaclsCommand(command string) WindowsPermissionsOption {
	return func(p *WindowsPermissions) error {
		p.IcaclsCommand = command
		return nil
	}
}

// WithRemoveDefaultPermissions removes the default permissions from the file.
func WithRemoveDefaultPermissions() WindowsPermissionsOption {
	return func(p *WindowsPermissions) error {
		p.RemoveDefaultPermissions = true
		return nil
	}
}
