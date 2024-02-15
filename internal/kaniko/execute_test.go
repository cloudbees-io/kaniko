package kaniko

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
		ExecutablePath:              "/kaniko/executor",
		Dockerfile:                  "Dockerfile",
		DockerContext:               ".",
		Destination:                 "gcr.io/kaniko-project/executor:v1.6.0",
		Context:                     ctx,
		SkipDefaultRegistryFallback: true,
	}
	os.Setenv("DOCKER_BUILD_ARGS", "key1=value1,key2=value2")
	os.Setenv("DOCKER_LABELS", "key_l1=l_value1,key_l2=l_value2")
	os.Setenv("CLOUDBEES_OUTPUTS", "/tmp/fake-outputs")

	cmd, err := c.cmdBuilder("/tmp/kaniko-test-digest-file")
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
		"/tmp/kaniko-test-digest-file",
		"--skip-default-registry-fallback",
	}
	expectedCmd := exec.CommandContext(ctx, "/kaniko/executor", exepectedArgs...)

	require.Equal(t, expectedCmd.Args, cmd.Args)
}

func Test_writeActionOutput(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kaniko-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	fakeDigest := "sha256:cafebabebeef"
	digestFile := filepath.Join(tmpDir, "digest-file")
	err = os.WriteFile(digestFile, []byte(fakeDigest), 0640)
	require.NoError(t, err)

	for _, c := range []struct {
		name          string
		dest          string
		wantTag       string
		wantTagDigest string
		wantImage     string
	}{
		{
			name:          "empty dest",
			wantTag:       "latest",
			wantTagDigest: "latest@sha256:cafebabebeef",
			wantImage:     ":latest@sha256:cafebabebeef",
		},
		{
			name:          "single destination no tag",
			dest:          "my.registry/myimage",
			wantTag:       "latest",
			wantTagDigest: "latest@sha256:cafebabebeef",
			wantImage:     "my.registry/myimage:latest@sha256:cafebabebeef",
		},
		{
			name:          "single destination with tag",
			dest:          "my.registry/myimage:latest",
			wantTag:       "latest",
			wantTagDigest: "latest@sha256:cafebabebeef",
			wantImage:     "my.registry/myimage:latest@sha256:cafebabebeef",
		},
		{
			name:          "single destination with other tag",
			dest:          "my.registry/myimage:sometag",
			wantTag:       "sometag",
			wantTagDigest: "sometag@sha256:cafebabebeef",
			wantImage:     "my.registry/myimage:sometag@sha256:cafebabebeef",
		},
		{
			name:          "multiple destinations",
			dest:          "my.registry/myimage:sometag,my.registry/myimage:latest",
			wantTag:       "sometag",
			wantTagDigest: "sometag@sha256:cafebabebeef",
			wantImage:     "my.registry/myimage:sometag@sha256:cafebabebeef",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			var testee = Config{
				Destination: c.dest,
			}
			outDir, err := os.MkdirTemp("", "kaniko-test-")
			require.NoError(t, err)
			defer os.RemoveAll(outDir)

			err = testee.writeActionOutputs(outDir, digestFile)
			require.NoError(t, err, "write outputs")

			outputNames := []string{"digest", "tag", "tag-digest", "image"}
			expectValues := []string{fakeDigest, c.wantTag, c.wantTagDigest, c.wantImage}

			for i, outputName := range outputNames {
				v, err := os.ReadFile(filepath.Join(outDir, outputName))
				require.NoError(t, err, outputName)
				require.Equal(t, expectValues[i], string(v), outputName)
			}
		})
	}
}
