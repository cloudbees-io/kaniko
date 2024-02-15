package cmd

import (
	"context"
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
	cmd.Flags().StringVar(&cfg.Dockerfile, "dockerfile", "", "Dockerfile is the path to the Dockerfile to build")
	cmd.Flags().StringVar(&cfg.DockerContext, "context", "", "Context is the path to the build context")
	cmd.Flags().StringVar(&cfg.Destination, "destination", "", "Destination is the destination of the built image")
	cmd.Flags().BoolVar(&cfg.SkipDefaultRegistryFallback, "skipDefaultRegistryFallback", false, "Fail if image is not found on registry mirrors")
}
