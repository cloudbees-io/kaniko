package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/cloudbees-io/kaniko/internal/kaniko"
)

var (
	cmd = &cobra.Command{
		Use:   "kaniko-action",
		Short: "Build and push container images using Kaniko",
		Long:  "Build and push container images using Kaniko",
		RunE:  run,
	}
	cfg kaniko.Config
)

func Execute() error {
	return cmd.Execute()
}

func run(command *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown arguments: %v", args)
	}
	newContext, cancel := context.WithCancel(context.Background())
	osChannel := make(chan os.Signal, 1)
	signal.Notify(osChannel, os.Interrupt)
	go func() {
		<-osChannel
		cancel()
	}()

	return cfg.Run(newContext)
}

func init() {
	// Define flags for configuring the Kaniko build
	cmd.Flags().StringVar(&cfg.Dockerfile, "dockerfile", "", "Dockerfile is the path to the Dockerfile to build")
	cmd.Flags().StringVar(&cfg.DockerContext, "context", "", "Context is the path to the build context")
	cmd.Flags().StringVar(&cfg.Destination, "destination", "", "Destination is the destination of the built image")
	cmd.Flags().StringVar(&cfg.RegistryMirrors, "registry-mirrors", "", "Registry mirrors to find images")
	cmd.Flags().BoolVar(&cfg.SkipDefaultRegistryFallback, "skip-default-registry-fallback", false, "Fail if image is not found on registry mirrors")
	cmd.Flags().StringVar(&cfg.Verbosity, "verbosity", "debug", "Verbosity level of the Kaniko executor")
	cmd.Flags().StringVar(&cfg.Target, "target", "", "Target stage to build in a multi-stage Dockerfile")
	cmd.Flags().StringVar(&cfg.TarPath, "tar-path", "", "Path to save the image tar file (optional). If set, the image will be saved as a tar file.")
}
