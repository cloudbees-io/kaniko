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
		RegistryMirrors:             "mirror.gcr.io,mycompany-docker-virtual.jfrog.io",
		SkipDefaultRegistryFallback: true,
		Verbosity:                   "debug",
		Target:                      "final-stage",
	}
	os.Setenv("DOCKER_BUILD_ARGS", "key1=value1,key2=value2")
	os.Setenv("DOCKER_LABELS", "key_l1=l_value1,key_l2=l_value2")
	os.Setenv("CLOUDBEES_OUTPUTS", "/tmp/fake-outputs")
	defer os.Unsetenv("DOCKER_BUILD_ARGS")
	defer os.Unsetenv("DOCKER_LABELS")
	defer os.Unsetenv("CLOUDBEES_OUTPUTS")

	cmd, err := c.cmdBuilder("/tmp/kaniko-test-digest-file")
	require.NoError(t, err)

	expectedArgs := []string{
		"--ignore-path=/cloudbees/",
		"--verbosity=debug",
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
		"--registry-mirror",
		"mirror.gcr.io",
		"--registry-mirror",
		"mycompany-docker-virtual.jfrog.io",
		"--digest-file",
		"/tmp/kaniko-test-digest-file",
		"--skip-default-registry-fallback",
		"--target",
		"final-stage",
	}
	expectedCmd := exec.CommandContext(ctx, "/kaniko/executor", expectedArgs...)

	require.Equal(t, expectedCmd.Args, cmd.Args)
}

func Test_CmdRegistryMirrors(t *testing.T) {
	ctx := context.Background()
	t.Run("multiple registry mirrors", func(t *testing.T) {
		var c = Config{
			ExecutablePath:              "/kaniko/executor",
			Dockerfile:                  "Dockerfile",
			DockerContext:               ".",
			Destination:                 "gcr.io/kaniko-project/executor:v1.6.0",
			Context:                     ctx,
			RegistryMirrors:             "mirror.gcr.io,mycompany-docker-virtual.jfrog.io",
			SkipDefaultRegistryFallback: false,
			Verbosity:                   "debug",
			Target:                      "final-stage",
		}
		cmd, err := c.cmdBuilder("/tmp/kaniko-test-digest-file")
		require.NoError(t, err)

		expectedArgs := []string{
			"--ignore-path=/cloudbees/",
			"--verbosity=debug",
			"--dockerfile",
			"Dockerfile",
			"--context",
			".",
			"--destination",
			"gcr.io/kaniko-project/executor:v1.6.0",
			"--registry-mirror",
			"mirror.gcr.io",
			"--registry-mirror",
			"mycompany-docker-virtual.jfrog.io",
			"--digest-file",
			"/tmp/kaniko-test-digest-file",
			"--target", // Add target to expected arguments
			"final-stage",
		}
		expectedCmd := exec.CommandContext(ctx, "/kaniko/executor", expectedArgs...)
		require.Equal(t, expectedCmd.Args, cmd.Args)
	})

	t.Run("no registry mirrors", func(t *testing.T) {
		var c = Config{
			ExecutablePath:              "/kaniko/executor",
			Dockerfile:                  "Dockerfile",
			DockerContext:               ".",
			Destination:                 "gcr.io/kaniko-project/executor:v1.6.0",
			RegistryMirrors:             "",
			SkipDefaultRegistryFallback: false,
			Context:                     ctx,
			Verbosity:                   "debug",
			Target:                      "final-stage",
		}
		cmd, err := c.cmdBuilder("/tmp/kaniko-test-digest-file")
		require.NoError(t, err)

		expectedArgs := []string{
			"--ignore-path=/cloudbees/",
			"--verbosity=debug",
			"--dockerfile",
			"Dockerfile",
			"--context",
			".",
			"--destination",
			"gcr.io/kaniko-project/executor:v1.6.0",
			"--digest-file",
			"/tmp/kaniko-test-digest-file",
			"--target", // Add target to expected arguments
			"final-stage",
		}
		expectedCmd := exec.CommandContext(ctx, "/kaniko/executor", expectedArgs...)
		require.Equal(t, expectedCmd.Args, cmd.Args)
	})

	t.Run("skip default registry fallback false", func(t *testing.T) {
		var c = Config{
			ExecutablePath:              "/kaniko/executor",
			Dockerfile:                  "Dockerfile",
			DockerContext:               ".",
			Destination:                 "gcr.io/kaniko-project/executor:v1.6.0",
			SkipDefaultRegistryFallback: false,
			Context:                     ctx,
			Verbosity:                   "debug",
			Target:                      "final-stage",
		}
		cmd, err := c.cmdBuilder("/tmp/kaniko-test-digest-file")
		require.NoError(t, err)

		expectedArgs := []string{
			"--ignore-path=/cloudbees/",
			"--verbosity=debug",
			"--dockerfile",
			"Dockerfile",
			"--context",
			".",
			"--destination",
			"gcr.io/kaniko-project/executor:v1.6.0",
			"--digest-file",
			"/tmp/kaniko-test-digest-file",
			"--target", // Add target to expected arguments
			"final-stage",
		}

		expectedCmd := exec.CommandContext(ctx, "/kaniko/executor", expectedArgs...)
		require.Equal(t, expectedCmd.Args, cmd.Args)
	})

}

func Test_CmdRegistryMaps(t *testing.T) {
	ctx := context.Background()
	t.Run("registry maps from context var 'cloudbees.registries'", func(t *testing.T) {
		var c = Config{
			ExecutablePath:              "/kaniko/executor",
			Dockerfile:                  "Dockerfile",
			DockerContext:               ".",
			Destination:                 "gcr.io/kaniko-project/executor:v1.6.0",
			Context:                     ctx,
			SkipDefaultRegistryFallback: false,
			Verbosity:                   "debug",
			Target:                      "final-stage",
		}

		require.NoError(t, os.Setenv("CLOUDBEES_REGISTRY_CONFIG", "testdata/registries.json"))
		defer os.Unsetenv("CLOUDBEES_REGISTRY_CONFIG")

		cmd, err := c.cmdBuilder("/tmp/kaniko-test-digest-file")
		require.NoError(t, err)

		expectedArgs := []string{
			"--ignore-path=/cloudbees/",
			"--verbosity=debug",
			"--dockerfile",
			"Dockerfile",
			"--context",
			".",
			"--destination",
			"gcr.io/kaniko-project/executor:v1.6.0",
			"--registry-map",
			"docker.io=mirror1.example.com/dockerhub;docker.io=mirror2.example.com/dockerhub;quay.io=mirror1.example.com/quay;quay.io=mirror2.example.com/quay",
			"--digest-file",
			"/tmp/kaniko-test-digest-file",
			"--target", // Add target to expected arguments
			"final-stage",
		}
		expectedCmd := exec.CommandContext(ctx, "/kaniko/executor", expectedArgs...)
		require.Equal(t, expectedCmd.Args, cmd.Args)
	})
	t.Run("no registry maps", func(t *testing.T) {
		var c = Config{
			ExecutablePath:              "/kaniko/executor",
			Dockerfile:                  "Dockerfile",
			DockerContext:               ".",
			Destination:                 "gcr.io/kaniko-project/executor:v1.6.0",
			Context:                     ctx,
			SkipDefaultRegistryFallback: false,
			Verbosity:                   "debug",
			Target:                      "final-stage",
		}

		cmd, err := c.cmdBuilder("/tmp/kaniko-test-digest-file")
		require.NoError(t, err)

		expectedArgs := []string{
			"--ignore-path=/cloudbees/",
			"--verbosity=debug",
			"--dockerfile",
			"Dockerfile",
			"--context",
			".",
			"--destination",
			"gcr.io/kaniko-project/executor:v1.6.0",
			"--digest-file",
			"/tmp/kaniko-test-digest-file",
			"--target", // Add target to expected arguments
			"final-stage",
		}
		expectedCmd := exec.CommandContext(ctx, "/kaniko/executor", expectedArgs...)
		require.Equal(t, expectedCmd.Args, cmd.Args)
	})
	t.Run("empty file in 'cloudbees.registries'", func(t *testing.T) {
		var c = Config{
			ExecutablePath:              "/kaniko/executor",
			Dockerfile:                  "Dockerfile",
			DockerContext:               ".",
			Destination:                 "gcr.io/kaniko-project/executor:v1.6.0",
			Context:                     ctx,
			SkipDefaultRegistryFallback: false,
			Verbosity:                   "debug",
			Target:                      "final-stage",
		}

		d, err := os.MkdirTemp("", "kaniko-test-")
		require.NoError(t, err)
		defer os.RemoveAll(d)

		require.NoError(t, os.WriteFile(filepath.Join(d, "registries.json"), []byte{}, 0644))
		require.NoError(t, os.Setenv("CLOUDBEES_REGISTRY_CONFIG", filepath.Join(d, "registries.json")))
		defer os.Unsetenv("CLOUDBEES_REGISTRY_CONFIG")

		cmd, err := c.cmdBuilder("/tmp/kaniko-test-digest-file")
		require.NoError(t, err)

		expectedArgs := []string{
			"--ignore-path=/cloudbees/",
			"--verbosity=debug",
			"--dockerfile",
			"Dockerfile",
			"--context",
			".",
			"--destination",
			"gcr.io/kaniko-project/executor:v1.6.0",
			"--digest-file",
			"/tmp/kaniko-test-digest-file",
			"--target", // Add target to expected arguments
			"final-stage",
		}
		expectedCmd := exec.CommandContext(ctx, "/kaniko/executor", expectedArgs...)
		require.Equal(t, expectedCmd.Args, cmd.Args)
	})
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
		{
			name:          "multiple destinations",
			dest:          "index.docker.io/kushalcp/my-sample-go-app:sometag, index.docker.io/urvashisingh/test-go-app:sometag",
			wantTag:       "sometag",
			wantTagDigest: "sometag@sha256:cafebabebeef",
			wantImage:     "index.docker.io/kushalcp/my-sample-go-app:sometag@sha256:cafebabebeef",
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
