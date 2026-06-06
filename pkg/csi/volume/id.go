package volume

import (
	"fmt"
	"path"
	"strings"
)

func ID(storageID, name string) string {
	return path.Join(storageID, name)
}

func Parse(volumeID string) (storageID, name string, err error) {
	parts := strings.Split(volumeID, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid volume id %q", volumeID)
	}
	return parts[0], parts[1], nil
}

func SanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

func ExportKey(volumeID string) string {
	return strings.NewReplacer("/", "_", " ", "_").Replace(volumeID)
}
