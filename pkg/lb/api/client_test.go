package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KubelanCloud/kks-provider-plugin/pkg/lb/provisioner"
)

func TestClientAllocate(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/loadbalancers" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(provisioner.LoadBalancer{
			ID: "abc",
			IP: "172.173.200.1",
		})
	}))
	t.Cleanup(srv.Close)

	client := NewClient(ClientConfig{BaseURL: srv.URL})
	lb, err := client.Allocate(context.Background(), provisioner.AllocateRequest{
		Namespace: "default",
		Name:      "web",
	})
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if lb.IP != "172.173.200.1" {
		t.Fatalf("unexpected lb: %#v", lb)
	}
}
