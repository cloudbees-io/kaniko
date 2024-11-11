package kaniko

import "context"

type Config struct {
	context.Context
	// ExecutablePath is the path to the Kaniko executor binary.
	ExecutablePath string
	// Dockerfile is the path to the Dockerfile to build.
	Dockerfile string `json:"dockerfile,omitempty"`
	// Context is the path to the build context.
	DockerContext string `json:"context,omitempty"`
	// Destination is the destination of the built image.
	Destination string `json:"destination,omitempty"`
	// RegistryMirrors contains registries used to pull images.
	RegistryMirrors string `json:"registryMirrors,omitempty"`
	// SkipDefaultRegistryFallback sets whether to use fallback if image isn't found in mirrors.
	SkipDefaultRegistryFallback bool `json:"skipDefaultRegistryFallback,omitempty"`
	// Verbosity is the verbosity level of the Kaniko executor.
	Verbosity string `json:"verbosity,omitempty"`
	// Target field allows you to build a particular stage in multistage docker files.
	Target string `json:"target,omitempty"`
	// TarPath is an optional path to save the image as a tar file.
	TarPath string `json:"tar-path,omitempty"`

	client HTTPClient
}

type Auth struct {
	// Auth is the base64 encoded credentials for the registry.
	Auth     string `json:"auth,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}
