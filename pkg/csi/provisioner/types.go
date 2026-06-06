package provisioner

import "context"

const (
	PublishContextLUN   = "scsi.lun"
	PublishContextSlot  = "proxmox.scsi_slot"
	PublishContextVolid = "proxmox.volid"
)

type ClusterInfo struct {
	StorageID string `json:"storage_id"`
}

type Volume struct {
	VolumeID      string            `json:"volume_id"`
	SizeBytes     int64             `json:"size_bytes"`
	VolumeContext map[string]string `json:"volume_context,omitempty"`
}

type CreateVolumeRequest struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
}

type PublishVolumeRequest struct {
	NodeID string `json:"node_id"`
}

type PublishVolumeResponse struct {
	PublishContext map[string]string `json:"publish_context"`
}

type UnpublishVolumeRequest struct {
	NodeID   string `json:"node_id"`
	VolumeID string `json:"volume_id"`
}

type VolumeExistsResponse struct {
	Exists bool `json:"exists"`
}

type Backend interface {
	ClusterInfo(ctx context.Context) (ClusterInfo, error)
	CreateVolume(ctx context.Context, req CreateVolumeRequest) (Volume, error)
	DeleteVolume(ctx context.Context, volumeID string) error
	VolumeExists(ctx context.Context, volumeID string) (bool, error)
	PublishVolume(ctx context.Context, volumeID, nodeID string) (PublishVolumeResponse, error)
	UnpublishVolume(ctx context.Context, volumeID, nodeID string) error
	Close() error
}
