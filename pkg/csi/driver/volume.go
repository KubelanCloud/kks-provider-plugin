package driver

import "github.com/KubelanCloud/kks-csi-plugin/pkg/csi/volume"

func sanitizeVolumeName(name string) string {
	return volume.SanitizeName(name)
}
