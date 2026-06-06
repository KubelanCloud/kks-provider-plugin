//go:build !linux

package driver

import (
	"fmt"

	mount "k8s.io/mount-utils"
	utilexec "k8s.io/utils/exec"
)

const defaultLinuxFsType = "ext4"

func newMounter() *mount.SafeFormatAndMount {
	return mount.NewSafeFormatAndMount(mount.New(""), utilexec.New())
}

func findDiskByLUN(lun int) (string, error) {
	return "", fmt.Errorf("scsi volume mount is only supported on linux (lun %d)", lun)
}

func formatAndMount(source, target, fsType string, options []string, mounter *mount.SafeFormatAndMount) error {
	return mounter.FormatAndMount(source, target, fsType, options)
}

func cleanupMountPoint(target string, mounter *mount.SafeFormatAndMount) error {
	return mount.CleanupMountPoint(target, mounter, true)
}

func bindMount(source, target string) error {
	m := newMounter()
	return m.Mount(source, target, "", []string{"bind"})
}

func isMounted(path string) bool {
	m := newMounter()
	notMnt, err := m.IsLikelyNotMountPoint(path)
	return err == nil && !notMnt
}
