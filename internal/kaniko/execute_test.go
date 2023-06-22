package kaniko

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_validateDockerConfigJson(t *testing.T) {
	t.Run("empty dockerConfigJson", func(t *testing.T) {
		var c = Config{}
		require.EqualError(t, c.validateDockerConfigJson(), "docker registry host and credentials is a required parameter, please set the DOCKERCONFIGJSON environment variable as per the action documentation")
	})
	t.Run("valid dockerConfigJson", func(t *testing.T) {
		var c = Config{
			DockerConfigJson: `{"auths":{"<registry host>":{"username":"<username>","password":"<password>","auth":"<username>:<password>"}}}`,
		}

		require.NoError(t, c.validateDockerConfigJson())
	})
}

func Test_processDestinations(t *testing.T) {
	t.Run("single destination", func(t *testing.T) {
		var c = Config{
			Destination: "gcr.io/kaniko-project/executor:v1.6.0",
		}

		require.Equal(t, []string{"gcr.io/kaniko-project/executor:v1.6.0"}, c.processDestinations())
	})
	t.Run("multiple destinations", func(t *testing.T) {
		var c = Config{
			Destination: "gcr.io/kaniko-project/executor:v1.6.0, gcr.io/kaniko-project/executor:v1.6.1",
		}

		require.Equal(t, []string{"gcr.io/kaniko-project/executor:v1.6.0", " gcr.io/kaniko-project/executor:v1.6.1"}, c.processDestinations())
	})
}

func Test_processBuildArgs(t *testing.T) {
	t.Run("single build arg", func(t *testing.T) {
		var c = Config{
			BuildArgs: "key1=value1",
		}

		require.Equal(t, []string{"key1=value1"}, c.processBuildArgs())
	})
	t.Run("multiple build args", func(t *testing.T) {
		var c = Config{
			BuildArgs: "key1=value1,key2=value2,key3='value3 with spaces'",
		}

		require.Equal(t, []string{"key1=value1", "key2=value2", "key3='value3 with spaces'"}, c.processBuildArgs())
	})
}

func Test_processLabels(t *testing.T) {
	t.Run("single label", func(t *testing.T) {
		var c = Config{
			Labels: "key1=value1",
		}

		require.Equal(t, []string{"key1=value1"}, c.processLabels())
	})
	t.Run("multiple labels", func(t *testing.T) {
		var c = Config{
			Labels: "key1=value1,key2=value2",
		}

		require.Equal(t, []string{"key1=value1", "key2=value2"}, c.processLabels())
	})
}

func Test_cmdBuilder(t *testing.T) {
	ctx := context.Background()

	var c = Config{
		ExecutablePath:   "/kaniko/executor",
		DockerConfigJson: `{"auths":{"<registry host>":{"username":"<username>","password":"<password>","auth":"<username>:<password>"}}}`,
		Dockerfile:       "Dockerfile",
		DockerContext:    ".",
		Destination:      "gcr.io/kaniko-project/executor:v1.6.0",
		BuildArgs:        "key1=value1,key2=value2",
		Labels:           "key_l1=l_value1,key_l2=l_value2",
		Context:          ctx,
	}

	cmd, err := c.cmdBuilder()
	require.NoError(t, err)

	exepectedArgs := []string{
		"--dockerfile",
		"Dockerfile",
		"--context",
		"\\.",
		"--destination",
		"gcr.io/kaniko-project/executor:v1.6.0",
		"--build-arg",
		"key1=value1",
		"--build-arg",
		"key2=value2",
		"--label",
		"key_l1=l_value1",
		"--label",
		"key_l2=l_value2",
	}
	expectedCmd := exec.CommandContext(ctx, "/kaniko/executor", exepectedArgs...)
	expectedCmd.Env = []string{
		"IFS=''",
	}
	
	require.Equal(t, expectedCmd.Args, cmd.Args)
	require.Equal(t, expectedCmd.Env, cmd.Env)
}
