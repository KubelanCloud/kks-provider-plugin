package volume

import "testing"

func TestParseVolumeID(t *testing.T) {
	storageID, name, err := Parse("nfs-prod/pvc-1")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if storageID != "nfs-prod" || name != "pvc-1" {
		t.Fatalf("unexpected parse result: %q %q", storageID, name)
	}
}

func TestParseVolumeIDRejectsInvalid(t *testing.T) {
	if _, _, err := Parse("only-one-part"); err == nil {
		t.Fatal("expected error for invalid volume id")
	}
}

func TestID(t *testing.T) {
	if got := ID("nfs-prod", "pvc-1"); got != "nfs-prod/pvc-1" {
		t.Fatalf("unexpected id: %q", got)
	}
}
