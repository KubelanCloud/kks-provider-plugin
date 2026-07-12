package driver

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/KubelanCloud/kks-provider-plugin/config"
	"github.com/KubelanCloud/kks-provider-plugin/pkg/csi/client"
	"github.com/KubelanCloud/kks-provider-plugin/pkg/csi/provisioner"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Driver struct {
	cfg        *config.Config
	log        *zap.SugaredLogger
	backend    provisioner.Backend
	storageID  string
	identity   *IdentityServer
	controller *ControllerServer
	node       *NodeServer
}

func Run(ctx context.Context, cfg *config.Config, logger *zap.Logger) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	log := logger.Sugar()

	backend := client.NewBackend(cfg.Client)
	defer backend.Close()

	cluster, err := backend.ClusterInfo(ctx)
	if err != nil {
		return fmt.Errorf("resolve storage info from csi server: %w", err)
	}

	d := &Driver{
		cfg:       cfg,
		log:       log,
		backend:   backend,
		storageID: cluster.StorageID,
		identity:  newIdentityServer(cfg),
	}

	switch cfg.Driver.Mode {
	case config.CSIModeController, config.CSIModeAll:
		d.controller = newControllerServer(d)
	case config.CSIModeNode:
	}

	switch cfg.Driver.Mode {
	case config.CSIModeNode, config.CSIModeAll:
		d.node = newNodeServer(d)
	case config.CSIModeController:
	}

	endpoint, err := parseEndpoint(cfg.Driver.Endpoint)
	if err != nil {
		return err
	}
	if err := ensureSocketDir(endpoint); err != nil {
		return err
	}

	listener, err := net.Listen(endpoint.network, endpoint.address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Driver.Endpoint, err)
	}
	defer listener.Close()

	server := grpc.NewServer()
	csi.RegisterIdentityServer(server, d.identity)
	if d.controller != nil {
		csi.RegisterControllerServer(server, d.controller)
	}
	if d.node != nil {
		csi.RegisterNodeServer(server, d.node)
	}

	log.Infof(
		"starting csi client name=%s endpoint=%s mode=%s node_id=%s server=%s storage_id=%s",
		cfg.Driver.Name,
		cfg.Driver.Endpoint,
		cfg.Driver.Mode,
		cfg.Driver.NodeID,
		cfg.Client.ServerURL,
		cluster.StorageID,
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down csi client")
		server.GracefulStop()
		return nil
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("csi grpc server stopped: %w", err)
		}
		return nil
	}
}

type endpoint struct {
	network string
	address string
}

func parseEndpoint(raw string) (endpoint, error) {
	if strings.HasPrefix(raw, "unix://") {
		return endpoint{
			network: "unix",
			address: strings.TrimPrefix(raw, "unix://"),
		}, nil
	}
	if strings.HasPrefix(raw, "tcp://") {
		return endpoint{
			network: "tcp",
			address: strings.TrimPrefix(raw, "tcp://"),
		}, nil
	}
	return endpoint{}, fmt.Errorf("unsupported endpoint %q (use unix:// or tcp://)", raw)
}

func ensureSocketDir(ep endpoint) error {
	if ep.network != "unix" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(ep.address), 0o755); err != nil {
		return fmt.Errorf("create socket directory: %w", err)
	}
	if err := os.Remove(ep.address); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale socket: %w", err)
	}
	return nil
}
