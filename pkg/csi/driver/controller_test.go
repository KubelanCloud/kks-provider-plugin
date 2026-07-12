package driver

import (
	"context"
	"testing"

	"github.com/KubelanCloud/kks-provider-plugin/pkg/csi/provisioner"
	"github.com/container-storage-interface/spec/lib/go/csi"
)

type fakeBackend struct {
	createReq provisioner.CreateVolumeRequest
}

func (f *fakeBackend) ClusterInfo(context.Context) (provisioner.ClusterInfo, error) {
	return provisioner.ClusterInfo{}, nil
}

func (f *fakeBackend) CreateVolume(_ context.Context, req provisioner.CreateVolumeRequest) (provisioner.Volume, error) {
	f.createReq = req
	return provisioner.Volume{VolumeID: "vol-1", SizeBytes: req.SizeBytes}, nil
}

func (f *fakeBackend) DeleteVolume(context.Context, string) error { return nil }

func (f *fakeBackend) VolumeExists(context.Context, string) (bool, error) { return false, nil }

func (f *fakeBackend) PublishVolume(context.Context, string, string) (provisioner.PublishVolumeResponse, error) {
	return provisioner.PublishVolumeResponse{}, nil
}

func (f *fakeBackend) UnpublishVolume(context.Context, string, string) error { return nil }

func (f *fakeBackend) Close() error { return nil }

func TestControllerCreateVolumeForwardsStorageClassParameters(t *testing.T) {
	t.Parallel()

	backend := &fakeBackend{}
	s := &ControllerServer{d: &Driver{backend: backend}}

	_, err := s.CreateVolume(context.Background(), &csi.CreateVolumeRequest{
		Name: "pvc-demo",
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: 10,
		},
		VolumeCapabilities: []*csi.VolumeCapability{{
			AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{}},
			AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER},
		}},
		Parameters: map[string]string{
			"kks.kloud/provisioning-mode": "deferred",
			"example":                     "value",
		},
	})
	if err != nil {
		t.Fatalf("CreateVolume returned error: %v", err)
	}

	if backend.createReq.Parameters["kks.kloud/provisioning-mode"] != "deferred" {
		t.Fatalf("unexpected forwarded parameters: %#v", backend.createReq.Parameters)
	}
}
