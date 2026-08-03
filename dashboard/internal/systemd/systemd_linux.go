//go:build linux

package systemd

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
)

const unitPath = "/etc/systemd/system/" + ServiceName

func ResolveIdentity() (Identity, error) {
	username := os.Getenv("SUDO_USER")
	var current *user.User
	var err error
	if username != "" {
		current, err = user.Lookup(username)
	} else {
		current, err = user.Current()
	}
	if err != nil {
		return Identity{}, fmt.Errorf("resolve runtime user: %w", err)
	}
	uid := current.Uid
	gid := current.Gid
	if username != "" {
		if sudoUID, sudoGID := os.Getenv("SUDO_UID"), os.Getenv("SUDO_GID"); sudoUID != "" && sudoGID != "" {
			uid = sudoUID
			gid = sudoGID
		}
	}
	group, err := user.LookupGroupId(gid)
	if err != nil {
		return Identity{}, fmt.Errorf("resolve runtime group: %w", err)
	}
	gids, err := current.GroupIds()
	if err != nil {
		return Identity{}, fmt.Errorf("resolve runtime groups: %w", err)
	}
	return Identity{Username: current.Username, Group: group.Name, UID: uid, GID: gid, GIDs: gids, Root: uid == "0"}, nil
}

func Install(options Options) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("install requires root privileges; run with sudo")
	}
	if err := prepareRuntimeDirectories(options); err != nil {
		return err
	}
	if err := validatePermissions(options); err != nil {
		return err
	}
	content := []byte(GenerateUnit(options))
	if existing, err := os.ReadFile(unitPath); err == nil {
		if err := os.WriteFile(unitPath+".previous", existing, 0o644); err != nil {
			return fmt.Errorf("back up existing systemd unit: %w", err)
		}
	}
	if err := os.WriteFile(unitPath, content, 0o644); err != nil {
		return fmt.Errorf("write systemd unit: %w", err)
	}
	if err := runSystemctl("daemon-reload"); err != nil {
		return err
	}
	if err := runSystemctl("enable", "--now", ServiceName); err != nil {
		return err
	}
	return nil
}

func prepareRuntimeDirectories(options Options) error {
	uid64, err := strconv.ParseUint(options.Identity.UID, 10, 32)
	if err != nil {
		return fmt.Errorf("parse runtime UID: %w", err)
	}
	gid64, err := strconv.ParseUint(options.Identity.GID, 10, 32)
	if err != nil {
		return fmt.Errorf("parse runtime GID: %w", err)
	}
	for _, target := range []string{filepath.Dir(options.Config.DashboardDatabase.Path), filepath.Dir(options.Config.Logging.File)} {
		if err := createDirectoryForIdentity(target, int(uid64), int(gid64)); err != nil {
			return fmt.Errorf("prepare runtime directory %s: %w", target, err)
		}
	}
	return nil
}

func createDirectoryForIdentity(target string, uid, gid int) error {
	var missing []string
	current := filepath.Clean(target)
	for {
		info, err := os.Stat(current)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("path exists but is not a directory")
			}
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			return fmt.Errorf("no existing parent directory")
		}
		current = parent
	}
	for i := len(missing) - 1; i >= 0; i-- {
		if err := os.Mkdir(missing[i], 0o750); err != nil && !os.IsExist(err) {
			return err
		}
		if err := os.Chown(missing[i], uid, gid); err != nil {
			return err
		}
	}
	return nil
}

func Uninstall() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("uninstall requires root privileges; run with sudo")
	}
	_ = runSystemctl("disable", "--now", ServiceName)
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove systemd unit: %w", err)
	}
	return runSystemctl("daemon-reload")
}

func validatePermissions(options Options) error {
	uid, _ := strconv.ParseUint(options.Identity.UID, 10, 32)
	gids := make(map[uint32]struct{}, len(options.Identity.GIDs))
	for _, value := range options.Identity.GIDs {
		gid, err := strconv.ParseUint(value, 10, 32)
		if err == nil {
			gids[uint32(gid)] = struct{}{}
		}
	}
	checks := []struct {
		path  string
		mask  uint32
		label string
	}{
		{options.BinaryPath, 1, "binary executable"},
		{options.Config.Path, 4, "configuration readable"},
		{filepath.Dir(options.Config.DashboardDatabase.Path), 3, "dashboard directory writable"},
		{filepath.Dir(options.Config.Logging.File), 3, "log directory writable"},
	}
	for _, check := range checks {
		if err := hasPermission(check.path, uint32(uid), gids, check.mask); err != nil {
			return fmt.Errorf("runtime user %s cannot use %s (%s): %w", options.Identity.Username, check.path, check.label, err)
		}
	}
	return nil
}

func hasPermission(path string, uid uint32, gids map[uint32]struct{}, mask uint32) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("unsupported file metadata")
	}
	if uid == 0 {
		return nil
	}
	mode := uint32(info.Mode().Perm())
	var allowed uint32
	if stat.Uid == uid {
		allowed = (mode >> 6) & 7
	} else if _, ok := gids[stat.Gid]; ok {
		allowed = (mode >> 3) & 7
	} else {
		allowed = mode & 7
	}
	if allowed&mask != mask {
		return fmt.Errorf("permission bits %o do not include %o", mode, mask)
	}
	return nil
}

func runSystemctl(args ...string) error {
	command := exec.Command("systemctl", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %v failed: %w: %s", args, err, output)
	}
	return nil
}
