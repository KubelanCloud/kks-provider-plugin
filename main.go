package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/KubelanCloud/kks-csi-plugin/config"
	"github.com/KubelanCloud/kks-csi-plugin/pkg/csi/driver"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync() //nolint:errcheck

	configPath := "csi.hcl"

	rootCmd := &cobra.Command{
		Use:   "kks-csi",
		Short: "Kloud CSI driver for user cluster nodes",
		Long:  "Runs the in-cluster CSI gRPC driver. Storage operations are sent to the kks management CSI server.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, configPath, logger)
		},
	}
	rootCmd.PersistentFlags().StringVarP(&configPath, "config-file", "c", "csi.hcl", "Path to driver config (optional when using env vars)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, configPath string, logger *zap.Logger) error {
	cfg, err := config.LoadClient(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, cancel := signalContext(cmd.Context())
	defer cancel()

	logger.Sugar().Infof("loaded csi driver config from %s", configPath)
	return driver.Run(ctx, cfg, logger)
}

func signalContext(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-ch:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}
