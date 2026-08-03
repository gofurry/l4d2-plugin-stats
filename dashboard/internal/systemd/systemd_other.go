//go:build !linux

package systemd

import "fmt"

func ResolveIdentity() (Identity, error) {
	return Identity{}, fmt.Errorf("systemd is only supported on Linux")
}
func Install(Options) error { return fmt.Errorf("systemd is only supported on Linux") }
func Uninstall() error      { return fmt.Errorf("systemd is only supported on Linux") }
