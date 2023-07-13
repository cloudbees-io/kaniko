package kaniko

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

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
	t.Run("no build arg", func(t *testing.T) {
		var c = Config{}
		os.Setenv("DOCKER_BUILD_ARGS", "")
		require.Nil(t, c.processBuildArgs())
	})
	t.Run("single build arg", func(t *testing.T) {
		var c = Config{}
		os.Setenv("DOCKER_BUILD_ARGS", "key1=value1")
		require.Equal(t, []string{"key1=value1"}, c.processBuildArgs())
	})
	t.Run("multiple build args", func(t *testing.T) {
		var c = Config{}
		os.Setenv("DOCKER_BUILD_ARGS", "key1=value1,key2=value2,key3='value3 with spaces'")
		require.Equal(t, []string{"key1=value1", "key2=value2", "key3='value3 with spaces'"}, c.processBuildArgs())
	})
}

func Test_processLabels(t *testing.T) {
	t.Run("no label", func(t *testing.T) {
		var c = Config{}
		os.Setenv("DOCKER_LABELS", "")
		require.Nil(t, c.processLabels())
	})
	t.Run("single label", func(t *testing.T) {
		var c = Config{}
		os.Setenv("DOCKER_LABELS", "key1=value1")
		require.Equal(t, []string{"key1=value1"}, c.processLabels())
	})
	t.Run("multiple labels", func(t *testing.T) {
		var c = Config{}
		os.Setenv("DOCKER_LABELS", "key1=value1,key2=value2")
		require.Equal(t, []string{"key1=value1", "key2=value2"}, c.processLabels())
	})
}

func Test_cmdBuilder(t *testing.T) {
	ctx := context.Background()

	var c = Config{
		ExecutablePath: "/kaniko/executor",
		Dockerfile:     "Dockerfile",
		DockerContext:  ".",
		Destination:    "gcr.io/kaniko-project/executor:v1.6.0",
		Context:        ctx,
	}
	os.Setenv("DOCKER_BUILD_ARGS", "key1=value1,key2=value2")
	os.Setenv("DOCKER_LABELS", "key_l1=l_value1,key_l2=l_value2")
	os.Setenv("CLOUDBEES_OUTPUTS", "/tmp/fake-outputs")

	cmd, err := c.cmdBuilder()
	require.NoError(t, err)

	exepectedArgs := []string{
		"--verbosity=debug",
		"--ignore-path=/cloudbees/",
		"--dockerfile",
		"Dockerfile",
		"--context",
		".",
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
		"--digest-file",
		"/tmp/fake-outputs/digest",
	}
	expectedCmd := exec.CommandContext(ctx, "/kaniko/executor", exepectedArgs...)

	require.Equal(t, expectedCmd.Args, cmd.Args)
}
