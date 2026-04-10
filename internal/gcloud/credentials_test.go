package gcloud

import (
	"errors"
	"testing"
)

func TestLoadServiceAccountCredentials(t *testing.T) {
	json := []byte(`{
  "type": "service_account",
  "project_id": "bishop-gopherbot",
  "client_email": "gopherbot-robot@example.com",
  "private_key": "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n"
}`)

	creds, err := LoadServiceAccountCredentials(func(path string) ([]byte, error) {
		if path != "gopherbot-key.json.enc" {
			t.Fatalf("path = %q, want %q", path, "gopherbot-key.json.enc")
		}
		return json, nil
	}, "gopherbot-key.json.enc")
	if err != nil {
		t.Fatalf("LoadServiceAccountCredentials() error: %v", err)
	}
	if creds.ProjectID != "bishop-gopherbot" {
		t.Fatalf("ProjectID = %q, want %q", creds.ProjectID, "bishop-gopherbot")
	}
	if creds.ClientEmail != "gopherbot-robot@example.com" {
		t.Fatalf("ClientEmail = %q, want %q", creds.ClientEmail, "gopherbot-robot@example.com")
	}
	if len(ServiceAccountClientOptions(creds)) != 1 {
		t.Fatalf("ServiceAccountClientOptions() len = %d, want 1", len(ServiceAccountClientOptions(creds)))
	}
}

func TestLoadServiceAccountCredentialsRejectsWrongType(t *testing.T) {
	_, err := LoadServiceAccountCredentials(func(path string) ([]byte, error) {
		return []byte(`{"type":"authorized_user","project_id":"p","client_email":"e","private_key":"k"}`), nil
	}, "creds.enc")
	if err == nil {
		t.Fatal("LoadServiceAccountCredentials() succeeded for wrong credential type")
	}
}

func TestLoadServiceAccountCredentialsPropagatesReadError(t *testing.T) {
	want := errors.New("boom")
	_, err := LoadServiceAccountCredentials(func(path string) ([]byte, error) {
		return nil, want
	}, "creds.enc")
	if !errors.Is(err, want) {
		t.Fatalf("LoadServiceAccountCredentials() error = %v, want wrapped %v", err, want)
	}
}

func TestResolveProjectID(t *testing.T) {
	got, err := ResolveProjectID("", &ServiceAccountCredentials{ProjectID: "bishop-gopherbot"})
	if err != nil {
		t.Fatalf("ResolveProjectID() error: %v", err)
	}
	if got != "bishop-gopherbot" {
		t.Fatalf("ResolveProjectID() = %q, want %q", got, "bishop-gopherbot")
	}

	got, err = ResolveProjectID("explicit-project", &ServiceAccountCredentials{ProjectID: "bishop-gopherbot"})
	if err != nil {
		t.Fatalf("ResolveProjectID(explicit) error: %v", err)
	}
	if got != "explicit-project" {
		t.Fatalf("ResolveProjectID(explicit) = %q, want %q", got, "explicit-project")
	}
}
