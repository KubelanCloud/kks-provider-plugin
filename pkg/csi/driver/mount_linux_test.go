//go:build linux

package driver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBindMountCreatesTargetAndIsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "globalmount")
	target := filepath.Join(root, "pods", "test", "mount")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "probe"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write probe file: %v", err)
	}

	mounter := newMounter()
	if err := mounter.Mount("tmpfs", source, "tmpfs", []string{}); err != nil {
		t.Fatalf("mount tmpfs at source: %v", err)
	}
	t.Cleanup(func() {
		_ = cleanupMountPoint(source, mounter)
	})

	if err := bindMount(source, target); err != nil {
		t.Fatalf("first bindMount failed: %v", err)
	}
	t.Cleanup(func() {
		_ = cleanupMountPoint(target, mounter)
	})

	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target mount path missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "probe")); err != nil {
		t.Fatalf("bind mount did not expose source contents: %v", err)
	}
	if err := bindMount(source, target); err != nil {
		t.Fatalf("second bindMount failed: %v", err)
	}
}
