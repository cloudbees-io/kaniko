package kaniko

import (
	"context"
	"fmt"
	"net/http"
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
		SendArtifactInfo:            true,
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
		"--send-artifact-info",
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

func Test_buildCreateArtifactInfoRequest(t *testing.T) {
	t.Run("destination is empty", func(t *testing.T) {
		var c = Config{}
		setOSEnv()

		_, err := c.buildCreateArtifactInfoRequest("", os.Getenv("CLOUDBEES_RUN_ID"), os.Getenv("CLOUDBEES_RUN_ATTEMPT"))
		require.EqualError(t, err, "destination is empty")
	})

	t.Run("invalid destination format - no name specified", func(t *testing.T) {
		var c = Config{}
		setOSEnv()
		destination := "gcr.io/kaniko-project/:v1.0"

		_, err := c.buildCreateArtifactInfoRequest(destination, os.Getenv("CLOUDBEES_RUN_ID"), os.Getenv("CLOUDBEES_RUN_ATTEMPT"))
		require.EqualErrorf(t, err, "failed to parse image reference 'gcr.io/kaniko-project/:v1.0': invalid reference format", "failed to parse image reference '%s': invalid reference format", destination)
	})

	t.Run("invalid destination format - no version specified", func(t *testing.T) {
		var c = Config{}
		setOSEnv()
		destination := "gcr.io/kaniko-project/executor:"

		_, err := c.buildCreateArtifactInfoRequest(destination, os.Getenv("CLOUDBEES_RUN_ID"), os.Getenv("CLOUDBEES_RUN_ATTEMPT"))
		require.EqualErrorf(t, err, "failed to parse image reference 'gcr.io/kaniko-project/executor:': invalid reference format", "failed to parse image reference '%s': invalid reference format", destination)
	})

	t.Run("success - named imgRef 1", func(t *testing.T) {
		var c = Config{}
		setOSEnv()
		destination := "ubuntu"

		artInfo, err := c.buildCreateArtifactInfoRequest(destination, os.Getenv("CLOUDBEES_RUN_ID"), os.Getenv("CLOUDBEES_RUN_ATTEMPT"))
		require.Nil(t, err)
		require.NotNil(t, artInfo)
		require.Equal(t, "ubuntu", artInfo["name"])
		require.Equal(t, "latest", artInfo["version"])
	})

	t.Run("success - tagged imgRef 2", func(t *testing.T) {
		var c = Config{}
		setOSEnv()
		destination := "myrepo/myimage:1."

		artInfo, err := c.buildCreateArtifactInfoRequest(destination, os.Getenv("CLOUDBEES_RUN_ID"), os.Getenv("CLOUDBEES_RUN_ATTEMPT"))
		require.Nil(t, err)
		require.NotNil(t, artInfo)
		require.Equal(t, "myrepo/myimage", artInfo["name"])
		require.Equal(t, "1.", artInfo["version"])
	})

	t.Run("success - tagged imgRef 3", func(t *testing.T) {
		var c = Config{}
		setOSEnv()
		destination := "public.ecr.aws/l7o7z1g8/actions/kaniko-action:a0cb1b7ee330e2f1ecf0e7bb974e167f30c0bce6"

		artInfo, err := c.buildCreateArtifactInfoRequest(destination, os.Getenv("CLOUDBEES_RUN_ID"), os.Getenv("CLOUDBEES_RUN_ATTEMPT"))
		require.Nil(t, err)
		require.NotNil(t, artInfo)
		require.Equal(t, "public.ecr.aws/l7o7z1g8/actions/kaniko-action", artInfo["name"])
		require.Equal(t, "a0cb1b7ee330e2f1ecf0e7bb974e167f30c0bce6", artInfo["version"])
	})

}

func setOSEnv() {
	os.Setenv("CLOUDBEES_API_URL", "https://cloudbees.io")
	os.Setenv("CLOUDBEES_API_TOKEN", "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUI")
	os.Setenv("CLOUDBEES_RUN_ID", "123e4567-e89b-12d3-a456-426614174000")
	os.Setenv("CLOUDBEES_RUN_ATTEMPT", "1")
}

type MockHTTPClient struct {
	Response *http.Response
	Err      error
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.Response, m.Err
}

func Test_createArtifactInfo(t *testing.T) {
	t.Run("client is nil", func(t *testing.T) {
		var c = Config{}
		setOSEnv()
		destinations := []string{"gcr.io/kaniko-project/executor:v1.6.0", " gcr.io/kaniko-project/executor:v1.6.1"}

		err := c.createArtifactInfo(nil, destinations)
		require.EqualError(t, err, "client is nil")
	})

	t.Run("destinations list empty", func(t *testing.T) {
		var c = Config{}
		setOSEnv()
		destinations := []string{}

		// Prepare a mock response
		mockResponse := &http.Response{}

		mockClient := &MockHTTPClient{
			Response: mockResponse,
			Err:      nil,
		}

		err := c.createArtifactInfo(mockClient, destinations)
		require.EqualError(t, err, "destinations is empty")
	})

	t.Run("CLOUDBEES_API_URL is empty", func(t *testing.T) {
		var c = Config{}
		setOSEnv()
		os.Setenv("CLOUDBEES_API_URL", "")

		destinations := []string{"gcr.io/kaniko-project/executor:v1.6.0", " gcr.io/kaniko-project/executor:v1.6.1"}

		// Prepare a mock response
		mockResponse := &http.Response{}

		mockClient := &MockHTTPClient{
			Response: mockResponse,
			Err:      nil,
		}

		err := c.createArtifactInfo(mockClient, destinations)
		require.EqualError(t, err, "failed to send artifact info to CloudBees Platform because of missing CLOUDBEES_API_URL environment variable")
	})

	t.Run("CLOUDBEES_API_TOKEN is nil", func(t *testing.T) {
		var c = Config{}
		setOSEnv()
		os.Setenv("CLOUDBEES_API_TOKEN", "")
		destinations := []string{"gcr.io/kaniko-project/executor:v1.6.0", " gcr.io/kaniko-project/executor:v1.6.1"}

		// Prepare a mock response
		mockResponse := &http.Response{}

		mockClient := &MockHTTPClient{
			Response: mockResponse,
			Err:      nil,
		}

		err := c.createArtifactInfo(mockClient, destinations)
		require.EqualError(t, err, "failed to send artifact info to CloudBees Platform because of missing CLOUDBEES_API_TOKEN environment variable")
	})

	t.Run("CLOUDBEES_RUN_ID is empty", func(t *testing.T) {
		var c = Config{}
		setOSEnv()
		os.Setenv("CLOUDBEES_RUN_ID", "")
		destinations := []string{"gcr.io/kaniko-project/executor:v1.6.0", " gcr.io/kaniko-project/executor:v1.6.1"}

		// Prepare a mock response
		mockResponse := &http.Response{}

		mockClient := &MockHTTPClient{
			Response: mockResponse,
			Err:      nil,
		}

		err := c.createArtifactInfo(mockClient, destinations)
		require.EqualError(t, err, "failed to send artifact info to CloudBees Platform because of missing CLOUDBEES_RUN_ID environment variable")
	})

	t.Run("CLOUDBEES_RUN_ATTEMPT is empty", func(t *testing.T) {
		var c = Config{}
		setOSEnv()
		os.Setenv("CLOUDBEES_RUN_ATTEMPT", "")
		destinations := []string{"gcr.io/kaniko-project/executor:v1.6.0", " gcr.io/kaniko-project/executor:v1.6.1"}

		// Prepare a mock response
		mockResponse := &http.Response{}

		mockClient := &MockHTTPClient{
			Response: mockResponse,
			Err:      nil,
		}

		err := c.createArtifactInfo(mockClient, destinations)
		require.EqualError(t, err, "failed to send artifact info because of missing CLOUDBEES_RUN_ATTEMPT environment variable")
	})

	t.Run("create - Success", func(t *testing.T) {
		var c = Config{}
		setOSEnv()
		destinations := []string{"gcr.io/kaniko-project/executor:v1.6.0", "gcr.io/kaniko-project/executor:v1.6.1"}

		// Prepare a mock response
		mockResponse := &http.Response{
			StatusCode: http.StatusOK,
			Body:       http.NoBody,
		}

		mockClient := &MockHTTPClient{
			Response: mockResponse,
			Err:      nil,
		}

		err := c.createArtifactInfo(mockClient, destinations)
		require.NoError(t, err)
	})

	t.Run("create - Error", func(t *testing.T) {
		var c = Config{}
		setOSEnv()
		destinations := []string{"gcr.io/kaniko-project/executor:v1.6.0", " gcr.io/kaniko-project/executor:v1.6.1"}

		mockClient := &MockHTTPClient{
			Response: nil,
			Err:      fmt.Errorf("network error"),
		}

		err := c.createArtifactInfo(mockClient, destinations)
		require.EqualError(t, err, "network error")
	})

	t.Run("create - statusCode not 200", func(t *testing.T) {
		var c = Config{}
		setOSEnv()
		destinations := []string{"gcr.io/kaniko-project/executor:v1.6.0", " gcr.io/kaniko-project/executor:v1.6.1"}

		// Prepare a mock response
		mockResponse := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       http.NoBody,
		}

		mockClient := &MockHTTPClient{
			Response: mockResponse,
			Err:      nil,
		}

		err := c.createArtifactInfo(mockClient, destinations)
		require.EqualErrorf(t, err, "failed to create artifact info: \nPOST https://cloudbees.io/v2/workflows/runs/artifactinfos\nHTTP/400 \n",
			"failed to create artifact info: \nPOST %s\nHTTP/%d %s\n",
			"https://cloudbees.io/v2/workflows/runs/artifactinfos", 400, "Bad Request")
	})
}
