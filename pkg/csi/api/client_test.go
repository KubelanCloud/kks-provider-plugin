package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KubelanCloud/kks-provider-plugin/pkg/csi/provisioner"
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
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var got provisioner.CreateVolumeRequest
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if got.Parameters["kks.kloud/provisioning-mode"] != "immediate" {
			t.Fatalf("unexpected parameters: %#v", got.Parameters)
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
		Parameters: map[string]string{
			"kks.kloud/provisioning-mode": "immediate",
		},
	})
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}
	if vol.VolumeID != "abc123/k8s-volumes/pvc-1" {
		t.Fatalf("unexpected volume: %#v", vol)
	}
}

func TestClientDeleteVolume(t *testing.T) {
	t.Parallel()

	const volumeID = "abc123/pvc-1"
	deleted := false

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/volumes/abc123/pvc-1":
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/volumes/missing/pvc-1":
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	client := NewClient(ClientConfig{BaseURL: srv.URL})
	if err := client.DeleteVolume(context.Background(), volumeID); err != nil {
		t.Fatalf("DeleteVolume failed: %v", err)
	}
	if !deleted {
		t.Fatal("expected delete request to reach backend")
	}
	if err := client.DeleteVolume(context.Background(), "missing/pvc-1"); err != nil {
		t.Fatalf("DeleteVolume should treat 404 as success: %v", err)
	}
}
