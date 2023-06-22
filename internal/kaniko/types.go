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
}

type DockerConfigJson struct {
	// Auths is a map of registries and their credentials.
	Auths map[string]Auth `json:"auths,omitempty"`
}

type Auth struct {
	// Auth is the base64 encoded credentials for the registry.
	Auth string `json:"auth,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}
