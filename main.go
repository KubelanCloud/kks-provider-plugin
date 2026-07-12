package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KubelanCloud/kks-provider-plugin/config"
	"github.com/KubelanCloud/kks-provider-plugin/pkg/csi/driver"
	"github.com/KubelanCloud/kks-provider-plugin/pkg/kloudlb/controller"
	"github.com/KubelanCloud/kks-provider-plugin/pkg/kloudlb/speaker"
	lbapi "github.com/KubelanCloud/kks-provider-plugin/pkg/lb/api"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync() //nolint:errcheck

	rootCmd := &cobra.Command{
		Use:   "kks-provider",
		Short: "Kloud provider plugin for user cluster nodes",
		Long:  "Runs the in-cluster provider components for CSI and LoadBalancer integration with kks management services.",
	}

	rootCmd.AddCommand(csiCmd(logger), lbControllerCmd(), lbSpeakerCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func csiCmd(logger *zap.Logger) *cobra.Command {
	configPath := "csi.hcl"
	cmd := &cobra.Command{
		Use:   "csi",
		Short: "Run the Kloud CSI driver",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCSI(cmd, configPath, logger)
		},
	}
	cmd.Flags().StringVarP(&configPath, "config-file", "c", "csi.hcl", "Path to driver config (optional when using env vars)")
	return cmd
}

func lbControllerCmd() *cobra.Command {
	var (
		apiURL string
		token  string
	)
	cmd := &cobra.Command{
		Use:   "lb-controller",
		Short: "Run the KloudLB Service controller",
		RunE: func(cmd *cobra.Command, args []string) error {
			apiURL = envOr(apiURL, "KLOUD_LB_API_URL")
			token = envOr(token, "KLOUD_LB_ACCESS_TOKEN")
			if apiURL == "" {
				return fmt.Errorf("KLOUD_LB_API_URL is required")
			}
			if token == "" {
				return fmt.Errorf("KLOUD_LB_ACCESS_TOKEN is required")
			}

			cfg, err := restConfig()
			if err != nil {
				return err
			}
			client, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				return err
			}

			lbClient := lbapi.NewClient(lbapi.ClientConfig{BaseURL: apiURL, Token: token, Timeout: 30 * time.Second})
			ctrl, err := controller.New(client, lbClient)
			if err != nil {
				return err
			}

			ctx, cancel := signalContext(cmd.Context())
			defer cancel()
			return ctrl.Run(ctx, 2)
		},
	}
	cmd.Flags().StringVar(&apiURL, "api-url", "", "KKS LoadBalancer API base URL")
	cmd.Flags().StringVar(&token, "access-token", "", "Cluster LB access token")
	return cmd
}

func lbSpeakerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lb-speaker",
		Short: "Run the KloudLB L2 speaker",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := restConfig()
			if err != nil {
				return err
			}
			client, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				return err
			}

			spk, err := speaker.New(client, speaker.NodeNameFromEnv(), speaker.InterfaceFromEnv())
			if err != nil {
				return err
			}

			ctx, cancel := signalContext(cmd.Context())
			defer cancel()
			return spk.Run(ctx)
		},
	}
	return cmd
}

func runCSI(cmd *cobra.Command, configPath string, logger *zap.Logger) error {
	cfg, err := config.LoadClient(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, cancel := signalContext(cmd.Context())
	defer cancel()

	logger.Sugar().Infof("loaded csi driver config from %s", configPath)
	return driver.Run(ctx, cfg, logger)
}

func restConfig() (*rest.Config, error) {
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = clientcmd.RecommendedHomeFile
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

func envOr(flagValue, envKey string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(envKey)
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
