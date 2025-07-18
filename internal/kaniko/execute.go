package kaniko

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudbees-io/registry-config/pkg/registries"
	"github.com/distribution/reference"
)

const (
	kanikoExecutorBinary = "executor"
)

// HTTPClient defines the methods that we need for our HTTP client.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type HttpClient struct {
	client *http.Client
}

func (r *HttpClient) Do(req *http.Request) (*http.Response, error) {
	return r.client.Do(req)
}

func (k *Config) Run(ctx context.Context) (err error) {
	k.Context = ctx
	k.client = &HttpClient{
		client: &http.Client{},
	}
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

	imageRef := ""
	if outDir != "" {
		imageRef, err = k.writeActionOutputs(outDir, digestFile)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Saving artifact information for the pushed images...\n")
	destinations := k.processDestinations()
	err = k.createArtifactInfo(destinations, imageRef)
	if err != nil {
		log.Printf("WARN: failed to save artifact information: %v", err)
	}
	return nil
}

func (k *Config) createArtifactInfo(destinations []string, imageRef string) error {

	if k.client == nil {
		return fmt.Errorf("client is nil")
	}

	if destinations == nil || len(destinations) == 0 {
		return fmt.Errorf("destinations is empty")
	}

	apiUrl := os.Getenv("CLOUDBEES_API_URL")
	if apiUrl == "" {
		return fmt.Errorf("missing CLOUDBEES_API_URL environment variable")
	}

	apiToken := os.Getenv("CLOUDBEES_API_TOKEN")
	if apiToken == "" {
		return fmt.Errorf("missing CLOUDBEES_API_TOKEN environment variable")
	}

	requestURL, err := url.JoinPath(apiUrl, "/v2/workflows/runs/artifactinfos")
	if err != nil {
		return err
	}

	runId := os.Getenv("CLOUDBEES_RUN_ID")
	if runId == "" {
		return fmt.Errorf("missing CLOUDBEES_RUN_ID environment variable")
	}

	runAttempt := os.Getenv("CLOUDBEES_RUN_ATTEMPT")
	if runAttempt == "" {
		return fmt.Errorf("missing CLOUDBEES_RUN_ATTEMPT environment variable")
	}

	// map artifact version IDs to their corresponding image destinations
	artifactVersionsResult := make(map[string]string, len(destinations))

	for _, destination := range destinations {
		destination = strings.TrimSpace(destination)
		fmt.Printf("Saving artifact information for image %v\n", destination)

		artifactInfo, err := k.buildCreateArtifactInfoRequest(destination, imageRef, runId, runAttempt)
		if err != nil {
			return err
		}

		artifactInfoBytes, err := json.Marshal(artifactInfo)
		if err != nil {
			return err
		}

		apiReq, err := http.NewRequest(
			"POST",
			requestURL,
			bytes.NewReader(artifactInfoBytes),
		)
		if err != nil {
			return err
		}

		apiReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiToken))
		apiReq.Header.Set("Content-Type", "application/json")
		apiReq.Header.Set("Accept", "application/json")

		resp, err := k.client.Do(apiReq)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()

		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		resBody := string(responseBody)

		if resp.StatusCode != 200 {
			return fmt.Errorf("request failed: \nPOST: %s\nHTTP/%d %s\nBODY: %s", requestURL, resp.StatusCode, resp.Status, resBody)
		} else {
			fmt.Printf("Saved artifact information for image %v\n", destination)

			jsonResponseMap := make(map[string]interface{})
			err = json.Unmarshal(responseBody, &jsonResponseMap)
			if err != nil {
				return err
			}

			if id, ok := jsonResponseMap["id"].(string); ok {
				// Store the artifact ID in the result map
				artifactVersionsResult[destination] = id
			} else {
				return fmt.Errorf("unexpected response format: missing 'id' field in response for destination %s", destination)
			}
		}
	}

	err = writeArtifactIdsAsOutput(artifactVersionsResult)
	if err != nil {
		return err
	}

	return nil
}

func writeArtifactIdsAsOutput(value map[string]string) error {
	outputsDir := os.Getenv("CLOUDBEES_OUTPUTS")
	if outputsDir == "" {
		return fmt.Errorf("CLOUDBEES_OUTPUTS environment variable missing")
	}

	outputBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	outputFile := filepath.Join(outputsDir, "artifact-ids")
	err = os.WriteFile(outputFile, outputBytes, 0640)
	if err != nil {
		return fmt.Errorf("failed to write to %s: %w", outputFile, err)
	}
	fmt.Printf("Output parameter '%s' value %v written to %s\n", "artifact-ids", value, outputFile)
	return nil
}

// CreateArtifactInfoMap is a map of key-value pairs that is used to store CreateArtifactInfoRequest data
type CreateArtifactInfoMap map[string]interface{}

func (k *Config) buildCreateArtifactInfoRequest(destination, imageRef, runId, runAttempt string) (CreateArtifactInfoMap, error) {

	if destination == "" {
		return nil, fmt.Errorf("destination is empty")
	}

	ref, err := reference.Parse(destination)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image reference '%s': %w", destination, err)
	}

	var imgName, imgVer string
	// Check if the reference is a tagged or digest reference
	switch ref := ref.(type) {
	case reference.Tagged:
		namedRef := ref.(reference.Named)
		imgName = namedRef.Name()
		imgVer = ref.Tag()
	case reference.Digested:
		namedRef := ref.(reference.Named)
		imgName = namedRef.Name()
		imgVer = ref.Digest().String()
	case reference.Named:
		imgName = ref.Name()
		imgVer = "latest"
	default:
		return nil, fmt.Errorf("unsupported destination type: %T for destination: %s\n", ref, destination)
	}

	if imgName == "" || imgVer == "" {
		return nil, fmt.Errorf("failed to build kaniko artifact info request: invalid destination format, %s", destination)
	}

	imageDigest := ""
	if imageRef != "" {
		_, after, found := strings.Cut(imageRef, "@")
		if found {
			imageDigest = after
		}
	}
	artInfo := CreateArtifactInfoMap{
		"runId":       runId,
		"run_attempt": runAttempt,
		"name":        imgName,
		"version":     imgVer,
		"url":         destination,
		"type":        "docker",
		"digest":      imageDigest,
	}

	return artInfo, nil
}

func (k *Config) writeActionOutputs(outDir, digestFile string) (string, error) {
	dest := k.processDestinations()[0]
	tag := "latest"
	if pos := strings.Index(dest, ":"); pos > 0 && pos < len(dest)-1 {
		tag = dest[pos+1:]
		dest = dest[:pos]
	}
	digest, err := os.ReadFile(digestFile)
	if err != nil {
		return "", fmt.Errorf("read kaniko image digest: %w", err)
	}
	err = os.WriteFile(filepath.Join(outDir, "digest"), digest, 0640)
	if err != nil {
		return "", fmt.Errorf("write digest output: %w", err)
	}
	err = os.WriteFile(filepath.Join(outDir, "tag"), []byte(tag), 0640)
	if err != nil {
		return "", fmt.Errorf("write tag output: %w", err)
	}
	tagDigest := fmt.Sprintf("%s@%s", tag, string(digest))
	err = os.WriteFile(filepath.Join(outDir, "tag-digest"), []byte(tagDigest), 0640)
	if err != nil {
		return "", fmt.Errorf("write tag-digest output: %w", err)
	}
	// NOTE: if imageRef format is changed, NEED to update logic to fetch digest in buildCreateArtifactInfoRequest func
	imageRef := fmt.Sprintf("%s:%s@%s", dest, tag, string(digest))
	err = os.WriteFile(filepath.Join(outDir, "image"), []byte(imageRef), 0640)
	if err != nil {
		return "", fmt.Errorf("write image output: %w", err)
	}
	return imageRef, nil
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
		fmt.Printf("Targeted stage:%v", k.Target)
		cmdArgs = append(cmdArgs, "--target", k.Target)
	}

	if k.TarPath != "" {
		cmdArgs = append(cmdArgs, "--tar-path", k.TarPath)
	}

	kanikoCmd := exec.CommandContext(k.Context, k.ExecutablePath, cmdArgs...)
	kanikoCmd.Env = k.env()

	kanikoCmd.Stdout = os.Stdout
	kanikoCmd.Stderr = os.Stderr

	return kanikoCmd, nil
}
