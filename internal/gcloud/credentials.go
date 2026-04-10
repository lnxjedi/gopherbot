package gcloud

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/api/option"
)

type ServiceAccountCredentials struct {
	JSON        []byte
	ProjectID   string
	ClientEmail string
}

type serviceAccountJSON struct {
	Type        string `json:"type"`
	ProjectID   string `json:"project_id"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
}

func LoadServiceAccountCredentials(readEncryptedFile func(string) ([]byte, error), path string) (*ServiceAccountCredentials, error) {
	if readEncryptedFile == nil {
		return nil, fmt.Errorf("readEncryptedFile callback is required")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("credentials file path is required")
	}

	plaintext, err := readEncryptedFile(path)
	if err != nil {
		return nil, err
	}

	var parsed serviceAccountJSON
	if err := json.Unmarshal(plaintext, &parsed); err != nil {
		return nil, fmt.Errorf("parsing decrypted Google credentials JSON: %w", err)
	}
	if parsed.Type != "service_account" {
		return nil, fmt.Errorf("unsupported Google credential type %q; expected %q", parsed.Type, "service_account")
	}
	if strings.TrimSpace(parsed.ProjectID) == "" {
		return nil, fmt.Errorf("Google service account credentials missing project_id")
	}
	if strings.TrimSpace(parsed.ClientEmail) == "" {
		return nil, fmt.Errorf("Google service account credentials missing client_email")
	}
	if strings.TrimSpace(parsed.PrivateKey) == "" {
		return nil, fmt.Errorf("Google service account credentials missing private_key")
	}

	return &ServiceAccountCredentials{
		JSON:        plaintext,
		ProjectID:   parsed.ProjectID,
		ClientEmail: parsed.ClientEmail,
	}, nil
}

func ResolveProjectID(explicitProjectID string, creds *ServiceAccountCredentials) (string, error) {
	projectID := strings.TrimSpace(explicitProjectID)
	if projectID != "" {
		return projectID, nil
	}
	if creds != nil && strings.TrimSpace(creds.ProjectID) != "" {
		return strings.TrimSpace(creds.ProjectID), nil
	}
	return "", fmt.Errorf("Google project ID is required")
}

func ServiceAccountClientOptions(creds *ServiceAccountCredentials) []option.ClientOption {
	if creds == nil || len(creds.JSON) == 0 {
		return nil
	}
	return []option.ClientOption{option.WithAuthCredentialsJSON(option.ServiceAccount, creds.JSON)}
}
