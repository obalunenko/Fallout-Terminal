package tunnel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	policyDirectoryPattern = "fallout-terminal-ngrok-*"
	policyFilename         = "traffic-policy.json"
)

// PolicyFile is short-lived credential material and its registered cleanup.
// Cleanup is safe to call repeatedly from success, failure, and shutdown paths.
type PolicyFile struct {
	Path string

	directory  string
	cleanup    sync.Once
	cleanupErr error
}

// CreatePolicy writes a private ngrok traffic-policy file containing Basic
// Auth protection. JSON encoding is used so credentials are never interpolated
// into policy syntax.
func CreatePolicy(parent string, credentials Credentials) (*PolicyFile, error) {
	if err := validateCredentials(credentials); err != nil {
		return nil, err
	}
	if parent == "" {
		parent = os.TempDir()
	}

	directory, err := os.MkdirTemp(parent, policyDirectoryPattern)
	if err != nil {
		return nil, fmt.Errorf("prepare ngrok policy directory: %w", err)
	}
	policy := &PolicyFile{
		Path:      filepath.Join(directory, policyFilename),
		directory: directory,
	}
	removeOnError := true
	defer func() {
		if removeOnError {
			_ = policy.Cleanup()
		}
	}()

	if err := os.Chmod(directory, 0o700); err != nil {
		return nil, fmt.Errorf("protect ngrok policy directory: %w", err)
	}
	file, err := os.OpenFile(policy.Path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create ngrok policy: %w", err)
	}

	document := trafficPolicy{
		OnHTTPRequest: []trafficPolicyRule{{
			Actions: []trafficPolicyAction{{
				Type: "basic-auth",
				Config: trafficPolicyActionConfig{
					Realm:       "Fallout Terminal Players",
					Credentials: []string{credentials.Username + ":" + credentials.Password},
					Enforce:     true,
				},
			}},
		}},
	}
	encodeErr := json.NewEncoder(file).Encode(document)
	closeErr := file.Close()
	if encodeErr != nil {
		return nil, fmt.Errorf("encode ngrok policy: %w", encodeErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close ngrok policy: %w", closeErr)
	}
	if err := os.Chmod(policy.Path, 0o600); err != nil {
		return nil, fmt.Errorf("protect ngrok policy: %w", err)
	}

	removeOnError = false
	return policy, nil
}

// Cleanup removes the complete owned policy directory exactly once.
func (policy *PolicyFile) Cleanup() error {
	if policy == nil {
		return nil
	}
	policy.cleanup.Do(func() {
		policy.cleanupErr = os.RemoveAll(policy.directory)
	})
	return policy.cleanupErr
}

type trafficPolicy struct {
	OnHTTPRequest []trafficPolicyRule `json:"on_http_request"`
}

type trafficPolicyRule struct {
	Actions []trafficPolicyAction `json:"actions"`
}

type trafficPolicyAction struct {
	Type   string                    `json:"type"`
	Config trafficPolicyActionConfig `json:"config"`
}

type trafficPolicyActionConfig struct {
	Realm       string   `json:"realm"`
	Credentials []string `json:"credentials"`
	Enforce     bool     `json:"enforce"`
}
