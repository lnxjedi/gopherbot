package gcloud

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2/google"
)

func NewScopedHTTPClient(ctx context.Context, creds *ServiceAccountCredentials, scopes ...string) (*http.Client, error) {
	if creds == nil || len(creds.JSON) == 0 {
		return nil, fmt.Errorf("service account credentials are required")
	}
	cfg, err := google.JWTConfigFromJSON(creds.JSON, scopes...)
	if err != nil {
		return nil, fmt.Errorf("parsing service account credentials for scoped HTTP client: %w", err)
	}
	return cfg.Client(ctx), nil
}
