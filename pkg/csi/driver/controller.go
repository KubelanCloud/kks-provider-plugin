package driver

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/KubelanCloud/kks-csi-plugin/pkg/csi/provisioner"
)

type ControllerServer struct {
	d *Driver
}

func newControllerServer(d *Driver) *ControllerServer {
	return &ControllerServer{d: d}
}

func (s *ControllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	if req.GetName() == "" {
		return nil, invalidArgument("volume name is required")
	}
	if req.GetVolumeCapabilities() == nil {
		return nil, invalidArgument("volume capabilities are required")
	}

	capacity := int64(1 * 1024 * 1024 * 1024)
	if req.GetCapacityRange() != nil {
		if req.GetCapacityRange().GetRequiredBytes() > 0 {
			capacity = req.GetCapacityRange().GetRequiredBytes()
		} else if req.GetCapacityRange().GetLimitBytes() > 0 {
			capacity = req.GetCapacityRange().GetLimitBytes()
		}
	}

	vol, err := s.d.backend.CreateVolume(ctx, provisioner.CreateVolumeRequest{
		Name:      sanitizeVolumeName(req.GetName()),
		SizeBytes: capacity,
	})
	if err != nil {
		return nil, internalError(err)
	}

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      vol.VolumeID,
			CapacityBytes: vol.SizeBytes,
			VolumeContext: vol.VolumeContext,
		},
	}, nil
}

func (s *ControllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if req.GetVolumeId() == "" {
		return nil, invalidArgument("volume id is required")
	}

	if err := s.d.backend.DeleteVolume(ctx, req.GetVolumeId()); err != nil {
		return nil, internalError(err)
	}
	return &csi.DeleteVolumeResponse{}, nil
}

func (s *ControllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	if req.GetVolumeId() == "" {
		return nil, invalidArgument("volume id is required")
	}
	if req.GetNodeId() == "" {
		return nil, invalidArgument("node id is required")
	}

	pub, err := s.d.backend.PublishVolume(ctx, req.GetVolumeId(), req.GetNodeId())
	if err != nil {
		return nil, internalError(err)
	}

	return &csi.ControllerPublishVolumeResponse{
		PublishContext: pub.PublishContext,
	}, nil
}

func (s *ControllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	if req.GetVolumeId() == "" {
		return nil, invalidArgument("volume id is required")
	}
	nodeID := req.GetNodeId()
	if nodeID == "" {
		return &csi.ControllerUnpublishVolumeResponse{}, nil
	}

	if err := s.d.backend.UnpublishVolume(ctx, req.GetVolumeId(), nodeID); err != nil {
		return nil, internalError(err)
	}
	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

func (s *ControllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	_ = ctx
	if req.GetVolumeId() == "" {
		return nil, invalidArgument("volume id is required")
	}
	if req.GetVolumeCapabilities() == nil {
		return nil, invalidArgument("volume capabilities are required")
	}
	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: req.GetVolumeCapabilities(),
		},
	}, nil
}

func (s *ControllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	_ = ctx
	_ = req
	return nil, unimplemented("ListVolumes is not supported")
}

func (s *ControllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	_ = ctx
	_ = req
	return nil, unimplemented("GetCapacity is not supported")
}

func (s *ControllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	_ = ctx
	_ = req
	return nil, unimplemented("CreateSnapshot is not supported")
}

func (s *ControllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	_ = ctx
	_ = req
	return nil, unimplemented("DeleteSnapshot is not supported")
}

func (s *ControllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	_ = ctx
	_ = req
	return nil, unimplemented("ListSnapshots is not supported")
}

func (s *ControllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	_ = ctx
	_ = req
	return nil, unimplemented("ControllerExpandVolume is not supported")
}

func (s *ControllerServer) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	_ = ctx
	_ = req
	return nil, unimplemented("ControllerGetVolume is not supported")
}

func (s *ControllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	_ = ctx
	_ = req
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: []*csi.ControllerServiceCapability{
			controllerCapability(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME),
			controllerCapability(csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME),
		},
	}, nil
}

func (s *ControllerServer) ControllerModifyVolume(ctx context.Context, req *csi.ControllerModifyVolumeRequest) (*csi.ControllerModifyVolumeResponse, error) {
	_ = ctx
	_ = req
	return nil, unimplemented("ControllerModifyVolume is not supported")
}
