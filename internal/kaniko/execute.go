package kaniko

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	kanikoExecutorBinary   = "executor"
	dockerConfigJsonEnvVar = "DOCKERCONFIGJSON"
)

var (
	kanikoDir = "/kaniko"
)

func (k *Config) Run(ctx context.Context) (err error) {
	k.Context = ctx
	dockerConfigJson := os.Getenv(dockerConfigJsonEnvVar)

	if err := k.writeDockerConfigJson(dockerConfigJson); err != nil {
		return fmt.Errorf("dockerConfigJson validation failed: %w", err)
	}

	k.lookupBinary()

	kanikoCmd, err := k.cmdBuilder()
	if err != nil {
		return fmt.Errorf("failed to build kaniko command: %w", err)
	}

	fmt.Printf("Running command: %s\n", kanikoCmd.String())

	return kanikoCmd.Run()
}

func (k *Config) writeDockerConfigJson(dockerConfigJson string) error {
	if dockerConfigJson != "" {
		dockerConfigPath := filepath.Join(kanikoDir, ".docker", "config.json")

		log.Printf("writing docker config json to %s", dockerConfigPath)
		if err := os.MkdirAll(filepath.Dir(dockerConfigPath), 0700); err != nil {
			return fmt.Errorf("creating .docker directory: %w", err)
		}

		if err := json.Unmarshal([]byte(dockerConfigJson), &DockerConfigJson{}); err != nil {
			return fmt.Errorf("unmarshalling dockerConfigJson: %w", err)
		}

		// Write the docker config json into the KANIKO_DIR path
		if err := os.Setenv("KANIKO_DIR", kanikoDir); err != nil {
			return fmt.Errorf("setting KANIKO_DIR environment variable: %w", err)
		}

		if err := os.WriteFile(dockerConfigPath, []byte(dockerConfigJson), 0600); err != nil {
			return fmt.Errorf("writing docker config json: %w", err)
		}
	}
	return nil
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
	var cmdArgs []string

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

	cmdArgs = append(cmdArgs, "--verbosity", "debug")

	kanikoCmd := exec.CommandContext(k.Context, k.ExecutablePath, cmdArgs...)
	kanikoCmd.Env = k.env()

	kanikoCmd.Stdout = os.Stdout
	kanikoCmd.Stderr = os.Stderr

	return kanikoCmd, nil
}
