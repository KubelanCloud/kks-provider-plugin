package driver

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/KubelanCloud/kks-csi-plugin/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type IdentityServer struct {
	cfg *config.Config
}

func newIdentityServer(cfg *config.Config) *IdentityServer {
	return &IdentityServer{cfg: cfg}
}

func (s *IdentityServer) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	_ = ctx
	_ = req
	return &csi.GetPluginInfoResponse{
		Name:          s.cfg.Driver.Name,
		VendorVersion: "v0.1.0",
	}, nil
}

func (s *IdentityServer) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	_ = ctx
	_ = req
	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
		},
	}, nil
}

func (s *IdentityServer) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	_ = ctx
	_ = req
	return &csi.ProbeResponse{Ready: wrapperspb.Bool(true)}, nil
}

func controllerCapability(t csi.ControllerServiceCapability_RPC_Type) *csi.ControllerServiceCapability {
	return &csi.ControllerServiceCapability{
		Type: &csi.ControllerServiceCapability_Rpc{
			Rpc: &csi.ControllerServiceCapability_RPC{Type: t},
		},
	}
}

func nodeCapability(t csi.NodeServiceCapability_RPC_Type) *csi.NodeServiceCapability {
	return &csi.NodeServiceCapability{
		Type: &csi.NodeServiceCapability_Rpc{
			Rpc: &csi.NodeServiceCapability_RPC{Type: t},
		},
	}
}

func invalidArgument(msg string) error {
	return status.Error(codes.InvalidArgument, msg)
}

func notFound(msg string) error {
	return status.Error(codes.NotFound, msg)
}

func internalError(err error) error {
	return status.Errorf(codes.Internal, "%v", err)
}

func unimplemented(msg string) error {
	return status.Error(codes.Unimplemented, msg)
}
