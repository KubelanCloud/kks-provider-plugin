package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KubelanCloud/kks-csi-plugin/pkg/csi/provisioner"
)

func TestClientClusterInfo(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/cluster" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(provisioner.ClusterInfo{
			StorageID: "abc123",
		})
	}))
	t.Cleanup(srv.Close)

	client := NewClient(ClientConfig{BaseURL: srv.URL})
	info, err := client.ClusterInfo(context.Background())
	if err != nil {
		t.Fatalf("ClusterInfo failed: %v", err)
	}
	if info.StorageID != "abc123" {
		t.Fatalf("unexpected cluster info: %#v", info)
	}
}

func TestClientCreateVolume(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/volumes" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(provisioner.Volume{
			VolumeID:  "abc123/k8s-volumes/pvc-1",
			SizeBytes: 1024,
		})
	}))
	t.Cleanup(srv.Close)

	client := NewClient(ClientConfig{BaseURL: srv.URL})
	vol, err := client.CreateVolume(context.Background(), provisioner.CreateVolumeRequest{
		Name:      "pvc-1",
		SizeBytes: 1024,
	})
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}
	if vol.VolumeID != "abc123/k8s-volumes/pvc-1" {
		t.Fatalf("unexpected volume: %#v", vol)
	}
}
