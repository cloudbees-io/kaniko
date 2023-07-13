package kaniko

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	kanikoExecutorBinary = "executor"
)

func (k *Config) Run(ctx context.Context) (err error) {
	k.Context = ctx

	k.lookupBinary()

	kanikoCmd, err := k.cmdBuilder()
	if err != nil {
		return fmt.Errorf("failed to build kaniko command: %w", err)
	}

	fmt.Printf("Running command: %s\n", kanikoCmd.String())

	return kanikoCmd.Run()
}

func (k *Config) processDestinations() []string {
	return strings.Split(k.Destination, ",")
}

func (k *Config) processBuildArgs() []string {
	args := os.Getenv("DOCKER_BUILD_ARGS")
	if args == "" {
		return nil
	}
	return strings.Split(args, ",")
}

func (k *Config) processLabels() []string {
	labels := os.Getenv("DOCKER_LABELS")
	if labels == "" {
		return nil
	}
	return strings.Split(labels, ",")
}

func (k *Config) lookupBinary() {
	// The kaniko binary which executes the docker build and publish is called 'executor',
	// which is in the path '/kaniko/executor'.
	// Ref: https://github.com/GoogleContainerTools/kaniko/blob/main/deploy/Dockerfile
	execPath, err := exec.LookPath(kanikoExecutorBinary)
	if err != nil {
		log.Fatal("cannot find kaniko executor binary")
	}
	log.Printf("found kaniko executor binary at %s", execPath)
	k.ExecutablePath = execPath
}

func (k *Config) env() []string {
	return os.Environ()
}

func (k *Config) cmdBuilder() (*exec.Cmd, error) {
	cmdArgs := []string{
		"--verbosity=debug",
		"--ignore-path=/cloudbees/",
	}

	if k.Dockerfile != "" {
		cmdArgs = append(cmdArgs, "--dockerfile", k.Dockerfile)
	}

	if k.DockerContext != "" {
		cmdArgs = append(cmdArgs, "--context", k.DockerContext)
	}

	for _, destination := range k.processDestinations() {
		cmdArgs = append(cmdArgs, "--destination", destination)
	}

	for _, buildArg := range k.processBuildArgs() {
		cmdArgs = append(cmdArgs, "--build-arg", buildArg)
	}

	for _, label := range k.processLabels() {
		cmdArgs = append(cmdArgs, "--label", label)
	}

	if outDir := os.Getenv("CLOUDBEES_OUTPUTS"); outDir != "" {
		cmdArgs = append(cmdArgs, "--digest-file", filepath.Join(outDir, "digest"))
	}

	kanikoCmd := exec.CommandContext(k.Context, k.ExecutablePath, cmdArgs...)
	kanikoCmd.Env = k.env()

	kanikoCmd.Stdout = os.Stdout
	kanikoCmd.Stderr = os.Stderr

	return kanikoCmd, nil
}
