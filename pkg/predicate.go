// Copyright The SLSA team.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pkg

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	_ "github.com/sigstore/cosign/pkg/providers/all"
)

const (
	defaultFulcioAddr   = "https://v1.fulcio.sigstore.dev"
	defaultOIDCIssuer   = "https://oauth2.sigstore.dev/auth"
	defaultOIDCClientID = "sigstore"
	defaultRekorAddr    = "https://rekor.sigstore.dev"
)

// https://docs.github.com/en/actions/learn-github-actions/contexts#github-context.
type gitHubContext struct {
	Repository   string      `json:"repository"`
	ActionPath   string      `json:"action_path"`
	Workflow     string      `json:"workflow"`
	EventName    string      `json:"event_name"`
	EventPayload interface{} `json:"event"`
	SHA          string      `json:"sha"`
	RefType      string      `json:"ref_type"`
	Ref          string      `json:"ref"`
	BaseRef      string      `json:"base_ref"`
	HeadRef      string      `json:"head_ref"`
	Actor        string      `json:"actor"`
	RunNumber    string      `json:"run_number"`
	ServerUrl    string      `json:"server_url"`
	RunID        string      `json:"run_id"`
	RunAttempt   string      `json:"run_attempt"`
	// TODO: try removing this token:
	// `omitting Token from the struct causes an unexpected end of line from encoding/json`
	Token string `json:"token,omitempty"`
}

var (
	parametersVersion  int = 1
	buildConfigVersion int = 1
)

const (
	requestTokenEnvKey = "ACTIONS_ID_TOKEN_REQUEST_TOKEN"
	requestURLEnvKey   = "ACTIONS_ID_TOKEN_REQUEST_URL"
	audience           = "laurentsimon/slsa-github-generator/builder"
)

type (
	Step struct {
		Command []string `json:"command"`
		Env     []string `json:"env"`
	}
	BuildConfig struct {
		Version int    `json:"version"`
		Steps   []Step `json:"steps"`
	}

	Parameters struct {
		Version      int         `json:"version"`
		EventName    string      `json:"event_name"`
		EventPayload interface{} `json:"event_payload"`
		RefType      string      `json:"ref_type"`
		Ref          string      `json:"ref"`
		BaseRef      string      `json:"base_ref"`
		HeadRef      string      `json:"head_ref"`
		Actor        string      `json:"actor"`
		SHA1         string      `json:"sha1"`
	}
)

// GeneratePredicate translates github context into a SLSA predicate
// attestation.
// Spec: https://slsa.dev/provenance/v0.1
func GeneratePredicate(name, digest, ghContext, command, envs string) ([]byte, error) {
	gh := &gitHubContext{}

	if err := json.Unmarshal([]byte(ghContext), gh); err != nil {
		return nil, err
	}
	gh.Token = ""

	if _, err := hex.DecodeString(digest); err != nil || len(digest) != 64 {
		return nil, fmt.Errorf("sha256 digest is not valid: %s", digest)
	}

	com, err := unmarshallList(command)
	if err != nil {
		return nil, err
	}

	env, err := unmarshallList(envs)
	if err != nil {
		return nil, err
	}

	builderID, err := getReusableWorkflowID()
	if err != nil {
		return nil, err
	}

	predicate := slsa.ProvenancePredicate{
		// Identifies that this is a slsa-framework's slsa-github-generator-go' build.
		BuildType: "https://github.com/slsa-framework/slsa-github-generator-go@v1",
		// Identifies the reusable workflow and matches the job_workflow_ref.
		Builder: slsa.ProvenanceBuilder{
			// TODO(https://github.com/slsa-framework/slsa-github-generator-go/issues/6): add
			// version and hash.
			ID: fmt.Sprintf("https://github.com/%s", builderID),
		},
		Invocation: slsa.ProvenanceInvocation{
			ConfigSource: slsa.ConfigSource{
				EntryPoint: gh.Workflow,
				URI:        fmt.Sprintf("git+%s%s@%s.git", gh.ServerUrl, gh.Repository, gh.Ref),
				Digest: slsa.DigestSet{
					"sha1": gh.SHA,
				},
			},
			// Non user-controllable environment vars needed to reproduce the build.
			Environment: map[string]interface{}{
				"arch":               "amd64", // TODO: Does GitHub run actually expose this?
				"os":                 "ubuntu",
				"github_event_name":  gh.EventName,
				"github_run_number":  gh.RunNumber,
				"github_run_id":      gh.RunID,
				"github_run_attempt": gh.RunAttempt,
			},
			// Parameters coming from the trigger event.
			Parameters: Parameters{
				Version:      parametersVersion,
				EventName:    gh.EventName,
				Ref:          gh.Ref,
				BaseRef:      gh.BaseRef,
				HeadRef:      gh.HeadRef,
				RefType:      gh.RefType,
				Actor:        gh.Actor,
				SHA1:         gh.SHA,
				EventPayload: gh.EventPayload,
			},
		},
		BuildConfig: BuildConfig{
			Version: buildConfigVersion,
			Steps: []Step{
				// Single step.
				{
					Command: com,
					Env:     env,
				},
			},
		},
		Materials: []slsa.ProvenanceMaterial{
			{
				URI: fmt.Sprintf("git+%s.git", gh.Repository),
				Digest: slsa.DigestSet{
					"sha1": gh.SHA,
				},
			},
		},
	}

	attBytes, err := json.Marshal(predicate)
	if err != nil {
		return nil, err
	}

	return attBytes, nil
}

func unmarshallList(arg string) ([]string, error) {
	var res []string
	// If argument is empty, return an empty list early,
	// because `json.Unmarshal` would fail.
	if arg == "" {
		return res, nil
	}

	cs, err := base64.StdEncoding.DecodeString(arg)
	if err != nil {
		return res, fmt.Errorf("base64.StdEncoding.DecodeString: %w", err)
	}

	if err := json.Unmarshal(cs, &res); err != nil {
		return []string{}, fmt.Errorf("json.Unmarshal: %w", err)
	}
	return res, nil
}

// Note: see https://github.com/sigstore/cosign/blob/739947de3d0197fbaab926bd9b896963ebf47a19/pkg/providers/github/github.go.
func getReusableWorkflowID() (string, error) {
	urlKey := os.Getenv(requestURLEnvKey)
	if urlKey == "" {
		return "", fmt.Errorf("requestURLEnvKey is empty")
	}

	url := urlKey + "&audience=" + audience
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", "bearer "+os.Getenv(requestTokenEnvKey))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var payload struct {
		Value string `json:"value"`
	}

	// Extract the value from JSON payload.
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&payload); err != nil {
		return "", err
	}

	// This is a JWT token with 3 parts.
	parts := strings.Split(payload.Value, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid jwt token: found %d parts", len(parts))
	}

	content := parts[1]

	// Base64-decode the content.
	token, err := base64.RawURLEncoding.DecodeString(content)
	if err != nil {
		return "", fmt.Errorf("base64.RawURLEncoding.DecodeString: %w", err)
	}

	// Extract fields from JSON payload.
	var oidc struct {
		JobWorkflowRef string `json:"job_workflow_ref"`
	}

	if err := json.Unmarshal(token, &oidc); err != nil {
		return "", fmt.Errorf("json.Unmarshal: %w", err)
	}

	if oidc.JobWorkflowRef == "" {
		return "", fmt.Errorf("job_workflow_ref is empty")
	}

	return oidc.JobWorkflowRef, nil
}
