package kaniko

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	kanikoExecutorBinary = "executor"
	kanikoConfigJson     = "/kaniko/.docker/config.json"
)

func (k *Config) Run(ctx context.Context) error {
	k.Context = ctx
	if err := k.validateDockerConfigJson(); err != nil {
		return fmt.Errorf("dockerConfigJson validation failed: %w", err)
	}

	k.lookupBinary()
	if err := k.authConfig(); err != nil {
		return fmt.Errorf("failed to write registry auth config: %w", err)
	}

	kanikoCmd, err := k.cmdBuilder()
	if err != nil {
		return fmt.Errorf("failed to build kaniko command: %w", err)
	}

	return kanikoCmd.Run() 
}

func (k *Config) validateDockerConfigJson() error {
	return json.Unmarshal([]byte(k.DockerConfigJson), &DockerConfigJson{})
}

func (k *Config) processDestinations() []string {
	return strings.Split(k.Destination, ",")
}

func (k *Config) processBuildArgs() []string {
	return strings.Split(k.BuildArgs, ",")
}

func (k *Config) processLabels() []string {
	return strings.Split(k.Labels, ",")
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

func (k *Config) authConfig() error {
	if k.DockerConfigJson != "" {
		return os.WriteFile(kanikoConfigJson, []byte(k.DockerConfigJson), 0600)
	}
	return nil
}

func (k *Config) env() []string {
	return []string{
		"IFS=''", // https://github.com/GoogleContainerTools/kaniko#flag---build-arg
	}
}

func (k *Config) cmdBuilder() (*exec.Cmd, error) {
	var cmdArgs []string

	if k.Dockerfile != "" {
		cmdArgs = append(cmdArgs, "--dockerfile", k.Dockerfile)
	}

	if k.DockerContext != "" {
		cmdArgs = append(cmdArgs, "--context", regexp.QuoteMeta(k.DockerContext))
	}

	for _, destination := range k.processDestinations() {
		cmdArgs = append(cmdArgs, "--destination", destination)
	}

	for _, buildArg := range k.processBuildArgs() {
		cmdArgs = append(cmdArgs, "--build-arg", regexp.QuoteMeta(buildArg))
	}

	for _, label := range k.processLabels() {
		cmdArgs = append(cmdArgs, "--label", regexp.QuoteMeta(label))
	}

	kanikoCmd := exec.CommandContext(k.Context, k.ExecutablePath, cmdArgs...)
	kanikoCmd.Env = k.env()

	kanikoCmd.Stdout = os.Stdout
	kanikoCmd.Stderr = os.Stderr

	return kanikoCmd, nil
}
