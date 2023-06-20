package kaniko

import (
	"encoding/json"
	"os/exec"
	"strings"
)

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

func (k *Config) kanikoExecutorBinaryPath() (string, error) {
	// The kaniko binary which executes the docker build and publish is called 'executor', 
	// which is in the path '/kaniko/executor'.
	// Ref: https://github.com/GoogleContainerTools/kaniko/blob/main/deploy/Dockerfile
	return exec.LookPath("executor")
}