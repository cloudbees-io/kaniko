package cmd

import (
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
  return nil
}

func init() {
  cmd.Flags().StringVar(&cfg.DockerConfigJson, "dockerconfigjson", "", "Dockerconfigjson is a stringified json containing the registry and registry credentials")
  cmd.Flags().StringVar(&cfg.Dockerfile, "dockerfile", "", "Dockerfile is the path to the Dockerfile to build")
  cmd.Flags().StringVar(&cfg.Context, "context", "", "Context is the path to the build context")
  cmd.Flags().StringVar(&cfg.Destination, "destination", "", "Destination is the destination of the built image")
  cmd.Flags().StringVar(&cfg.BuildArgs, "build-args", "", "BuildArgs are the args passed to the Docker build")
  cmd.Flags().StringVar(&cfg.Labels, "labels", "", "Labels are the labels passed to the Docker build")
}

