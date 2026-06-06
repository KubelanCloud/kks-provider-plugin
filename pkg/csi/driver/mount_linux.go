//go:build linux

package driver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	mount "k8s.io/mount-utils"
	utilexec "k8s.io/utils/exec"
)

const defaultLinuxFsType = "ext4"

func newMounter() *mount.SafeFormatAndMount {
	return mount.NewSafeFormatAndMount(mount.New(""), utilexec.New())
}

func scsiHostRescan() {
	scsiPath := "/sys/class/scsi_host/"
	entries, err := os.ReadDir(scsiPath)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := filepath.Join(scsiPath, entry.Name(), "scan")
		_ = os.WriteFile(name, []byte("- - -"), 0o666)
	}
}

func findDiskByLUN(lun int) (string, error) {
	scsiHostRescan()

	sysPath := "/sys/bus/scsi/devices"
	entries, err := os.ReadDir(sysPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", sysPath, err)
	}

	for _, entry := range entries {
		name := entry.Name()
		parts := strings.Split(name, ":")
		if len(parts) < 4 {
			continue
		}
		deviceLUN, err := strconv.Atoi(parts[3])
		if err != nil || deviceLUN != lun {
			continue
		}

		vendorPath := filepath.Join(sysPath, name, "vendor")
		vendorBytes, err := os.ReadFile(vendorPath)
		if err != nil {
			continue
		}
		vendor := strings.TrimSpace(string(vendorBytes))
		if vendor != "QEMU" && strings.ToUpper(vendor) != "MSFT" {
			continue
		}

		blockDir := filepath.Join(sysPath, name, "block")
		devices, err := os.ReadDir(blockDir)
		if err != nil || len(devices) == 0 {
			continue
		}
		devName := devices[0].Name()

		for _, devLinkPath := range []string{"/dev/disk/by-id/", "/dev/disk/by-path/"} {
			if link, err := findDiskLink(devLinkPath, devName); err == nil {
				return link, nil
			}
		}
		return "/dev/" + devName, nil
	}

	return "", fmt.Errorf("failed to find disk by lun %d", lun)
}

func findDiskLink(devLinkPath, devName string) (string, error) {
	entries, err := os.ReadDir(devLinkPath)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		linkPath := filepath.Join(devLinkPath, entry.Name())
		target, err := os.Readlink(linkPath)
		if err != nil {
			continue
		}
		if strings.HasSuffix(target, devName) {
			return linkPath, nil
		}
	}
	return "", fmt.Errorf("device %s not found under %s", devName, devLinkPath)
}

func formatAndMount(source, target, fsType string, options []string, mounter *mount.SafeFormatAndMount) error {
	return mounter.FormatAndMount(source, target, fsType, options)
}

func cleanupMountPoint(target string, mounter *mount.SafeFormatAndMount) error {
	return mount.CleanupMountPoint(target, mounter, true)
}

func bindMount(source, target string) error {
	if !isMounted(source) {
		return fmt.Errorf("staging path %s is not mounted", source)
	}
	if isMounted(target) {
		return nil
	}
	mounter := mount.New("")
	if err := mounter.Mount(source, target, "", []string{"bind"}); err != nil {
		return fmt.Errorf("bind mount %s -> %s: %w", source, target, err)
	}
	return nil
}

func isMounted(path string) bool {
	out, err := exec.Command("findmnt", "-n", path).CombinedOutput()
	return err == nil && strings.TrimSpace(string(out)) != ""
}
