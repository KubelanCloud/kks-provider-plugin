package driver

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/KubelanCloud/kks-csi-plugin/pkg/csi/provisioner"
)

type NodeServer struct {
	d *Driver
}

func newNodeServer(d *Driver) *NodeServer {
	return &NodeServer{d: d}
}

func (s *NodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	_ = ctx
	if req.GetVolumeId() == "" {
		return nil, invalidArgument("volume id is required")
	}
	if req.GetStagingTargetPath() == "" {
		return nil, invalidArgument("staging target path is required")
	}
	if isBlockVolume(req.GetVolumeCapability()) {
		return nil, unimplemented("block volumes are not supported")
	}

	lun, err := lunFromPublishContext(req.GetPublishContext())
	if err != nil {
		return nil, invalidArgument(err.Error())
	}

	device, err := findDiskByLUN(lun)
	if err != nil {
		return nil, internalError(err)
	}

	if err := os.MkdirAll(req.GetStagingTargetPath(), 0o755); err != nil {
		return nil, internalError(fmt.Errorf("create staging path: %w", err))
	}

	fsType := defaultLinuxFsType
	options := []string{}
	if mnt := req.GetVolumeCapability().GetMount(); mnt != nil {
		if mnt.FsType != "" {
			fsType = mnt.FsType
		}
		options = append(options, mnt.MountFlags...)
	}

	mounter := newMounter()
	if err := formatAndMount(device, req.GetStagingTargetPath(), fsType, options, mounter); err != nil {
		return nil, internalError(fmt.Errorf("format and mount %s: %w", device, err))
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

func (s *NodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	_ = ctx
	if req.GetVolumeId() == "" {
		return nil, invalidArgument("volume id is required")
	}
	if req.GetStagingTargetPath() == "" {
		return nil, invalidArgument("staging target path is required")
	}

	mounter := newMounter()
	if err := cleanupMountPoint(req.GetStagingTargetPath(), mounter); err != nil {
		return nil, internalError(err)
	}
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (s *NodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	_ = ctx
	if req.GetVolumeId() == "" {
		return nil, invalidArgument("volume id is required")
	}
	if req.GetTargetPath() == "" {
		return nil, invalidArgument("target path is required")
	}
	if req.GetStagingTargetPath() == "" {
		return nil, invalidArgument("staging target path is required")
	}

	if !isMounted(req.GetStagingTargetPath()) {
		return nil, internalError(fmt.Errorf("staging path %s is not mounted", req.GetStagingTargetPath()))
	}

	if err := bindMount(req.GetStagingTargetPath(), req.GetTargetPath()); err != nil {
		return nil, internalError(err)
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (s *NodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	_ = ctx
	if req.GetVolumeId() == "" {
		return nil, invalidArgument("volume id is required")
	}
	if req.GetTargetPath() == "" {
		return nil, invalidArgument("target path is required")
	}

	mounter := newMounter()
	if err := cleanupMountPoint(req.GetTargetPath(), mounter); err != nil {
		return nil, internalError(err)
	}
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (s *NodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	_ = ctx
	_ = req
	return nil, unimplemented("NodeGetVolumeStats is not supported")
}

func (s *NodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	_ = ctx
	_ = req
	return nil, unimplemented("NodeExpandVolume is not supported")
}

func (s *NodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	_ = ctx
	_ = req
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			nodeCapability(csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME),
		},
	}, nil
}

func (s *NodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	_ = ctx
	_ = req
	return &csi.NodeGetInfoResponse{
		NodeId: s.d.cfg.Driver.NodeID,
	}, nil
}

func lunFromPublishContext(publishContext map[string]string) (int, error) {
	lunStr := strings.TrimSpace(publishContext[provisioner.PublishContextLUN])
	if lunStr == "" {
		return 0, fmt.Errorf("publish context %q is required", provisioner.PublishContextLUN)
	}
	lun, err := strconv.Atoi(lunStr)
	if err != nil {
		return 0, fmt.Errorf("invalid publish context %q: %w", provisioner.PublishContextLUN, err)
	}
	return lun, nil
}

func isBlockVolume(capability *csi.VolumeCapability) bool {
	return capability != nil && capability.GetBlock() != nil
}
