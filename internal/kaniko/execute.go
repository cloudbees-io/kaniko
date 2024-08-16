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

	"github.com/cloudbees-io/registry-config/pkg/registries"
)

const (
	kanikoExecutorBinary = "executor"
	registryConfigBinary = "registry-config"
)

func (k *Config) Run(ctx context.Context) (err error) {
	k.Context = ctx

	k.lookupBinary()

	outDir := os.Getenv("CLOUDBEES_OUTPUTS")
	digestFile := ""
	if outDir != "" {
		digestFile = filepath.Join(os.TempDir(), "kaniko-image-digest")
	}

	kanikoCmd, err := k.cmdBuilder(digestFile)
	if err != nil {
		return fmt.Errorf("failed to build kaniko command: %w", err)
	}

	fmt.Printf("Running command: %s\n", kanikoCmd.String())

	err = kanikoCmd.Run()
	if err != nil {
		return fmt.Errorf("run kaniko: %w", err)
	}

	if outDir != "" {
		err = k.writeActionOutputs(outDir, digestFile)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *Config) writeActionOutputs(outDir, digestFile string) error {
	dest := k.processDestinations()[0]
	tag := "latest"
	if pos := strings.Index(dest, ":"); pos > 0 && pos < len(dest)-1 {
		tag = dest[pos+1:]
		dest = dest[:pos]
	}
	digest, err := os.ReadFile(digestFile)
	if err != nil {
		return fmt.Errorf("read kaniko image digest: %w", err)
	}
	err = os.WriteFile(filepath.Join(outDir, "digest"), digest, 0640)
	if err != nil {
		return fmt.Errorf("write digest output: %w", err)
	}
	err = os.WriteFile(filepath.Join(outDir, "tag"), []byte(tag), 0640)
	if err != nil {
		return fmt.Errorf("write tag output: %w", err)
	}
	tagDigest := fmt.Sprintf("%s@%s", tag, string(digest))
	err = os.WriteFile(filepath.Join(outDir, "tag-digest"), []byte(tagDigest), 0640)
	if err != nil {
		return fmt.Errorf("write tag-digest output: %w", err)
	}
	imageRef := fmt.Sprintf("%s:%s@%s", dest, tag, string(digest))
	err = os.WriteFile(filepath.Join(outDir, "image"), []byte(imageRef), 0640)
	if err != nil {
		return fmt.Errorf("write image output: %w", err)
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

func (k *Config) processRegistryMirrors() []string {
	if len(k.RegistryMirrors) == 0 {
		return []string{}
	}
	return strings.Split(k.RegistryMirrors, ",")
}

func registryMapsInConfig() (string, error) {
	regConfig := os.Getenv("CLOUDBEES_REGISTRY_CONFIG")
	if regConfig == "" {
		return "", nil
	}

	if _, err := os.Stat(regConfig); err != nil && os.IsNotExist(err) {
		return "", nil
	}

	b, err := os.ReadFile(regConfig)
	if err != nil {
		return "", fmt.Errorf("failed to read registry config file: %w", err)
	}

	if len(b) == 0 {
		return "", nil
	}

	var regs = registries.Config{}
	if err := json.Unmarshal(b, &regs); err != nil {
		return "", fmt.Errorf("failed to parse registry config file: %w", err)
	}

	var regmaps []string
	for _, registry := range regs.Registries {
		prefix := registry.Prefix
		for _, mirror := range registry.Mirrors {
			regmaps = append(regmaps, fmt.Sprintf("%s=%s", prefix, mirror))
		}
	}

	return strings.Join(regmaps, ";"), nil
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

func validateVerbosity(verbosity string) error {
	knownVerbosities := []string{"panic", "fatal", "error", "warn", "info", "debug", "trace"}
	for _, knownVerbosity := range knownVerbosities {
		if verbosity == knownVerbosity {
			return nil
		}
	}
	return fmt.Errorf("unknown verbosity level: %s", verbosity)
}

func (k *Config) cmdBuilder(digestFile string) (*exec.Cmd, error) {
	cmdArgs := []string{
		"--ignore-path=/cloudbees/",
	}

	if k.Verbosity != "" {
		k.Verbosity = strings.ToLower(k.Verbosity)
		if errVerbosity := validateVerbosity(k.Verbosity); errVerbosity != nil {
			return nil, errVerbosity
		}
		cmdArgs = append(cmdArgs, "--verbosity="+k.Verbosity)
	}

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

	for _, mirror := range k.processRegistryMirrors() {
		cmdArgs = append(cmdArgs, "--registry-mirror", mirror)
	}

	registryMaps, err := registryMapsInConfig()
	if err != nil {
		return nil, err
	}

	if registryMaps != "" {
		cmdArgs = append(cmdArgs, "--registry-map", registryMaps)
	}

	if digestFile != "" {
		cmdArgs = append(cmdArgs, "--digest-file", digestFile)
	}

	if k.SkipDefaultRegistryFallback {
		cmdArgs = append(cmdArgs, "--skip-default-registry-fallback")
	}

	if k.Target != "" {
		fmt.Printf("Test:%v", k.Target)
		cmdArgs = append(cmdArgs, "--target", k.Target)
	}

	kanikoCmd := exec.CommandContext(k.Context, k.ExecutablePath, cmdArgs...)
	kanikoCmd.Env = k.env()

	kanikoCmd.Stdout = os.Stdout
	kanikoCmd.Stderr = os.Stderr

	return kanikoCmd, nil
}
