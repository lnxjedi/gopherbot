package firestorebrain

import (
	"context"
	"encoding/json"
	"strings"

	"cloud.google.com/go/firestore"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lnxjedi/gopherbot/robot"
)

//------------------------------------------------------------------------------
//  A) Configuration Struct
//------------------------------------------------------------------------------
//
// This struct maps to the JSON fields in your service account key, plus
// optional config fields like "Collection". If "Collection" is empty,
// we'll default to "gopherbotMemories."

type gcpFireStoreConfig struct {
	// These fields match what you have in your BrainConfig:
	Type                    string `yaml:"Type"`
	ProjectID               string `yaml:"ProjectID"`
	PrivateKeyID            string `yaml:"PrivateKeyID"`
	PrivateKey              string `yaml:"PrivateKey"`
	ClientEmail             string `yaml:"ClientEmail"`
	ClientID                string `yaml:"ClientID"`
	AuthURI                 string `yaml:"AuthURI"`
	TokenURI                string `yaml:"TokenURI"`
	AuthProviderX509CertURL string `yaml:"AuthProviderx509CertURL"`
	ClientX509CertURL       string `yaml:"Clientx509CertURL"`
	UniverseDomain          string `yaml:"UniverseDomain"`

	// Optional: let the user specify a Firestore collection name
	// in the config. Otherwise, default to "gopherbotMemories".
	Collection string `yaml:"Collection"`
}

// memoryDoc is the Firestore document structure for each memory.
type memoryDoc struct {
	Content []byte `firestore:"content"`
}

//------------------------------------------------------------------------------
//  B) The Brain Struct implementing robot.SimpleBrain
//------------------------------------------------------------------------------

type fsBrain struct {
	handler robot.Handler
	config  gcpFireStoreConfig
	client  *firestore.Client
	ctx     context.Context
	cancel  context.CancelFunc
	colName string
}

//------------------------------------------------------------------------------
//  C) provider function (called by your static.go or plugin registration)
//------------------------------------------------------------------------------

func provider(r robot.Handler) robot.SimpleBrain {
	var cfg gcpFireStoreConfig
	// Gopherbot will parse the YAML/JSON config into cfg
	r.GetBrainConfig(&cfg)

	// If user didn't specify a collection, default to "gopherbotMemories"
	if cfg.Collection == "" {
		cfg.Collection = "gopherbotMemories"
	}

	// Create a context for Firestore usage
	ctx, cancel := context.WithCancel(context.Background())

	// Convert the escaped newlines in PrivateKey to real newlines
	privateKeyFixed := strings.ReplaceAll(cfg.PrivateKey, "\\n", "\n")

	// Build JSON credentials from the config fields
	credMap := map[string]string{
		"type":                        cfg.Type,
		"project_id":                  cfg.ProjectID,
		"private_key_id":              cfg.PrivateKeyID,
		"private_key":                 privateKeyFixed,
		"client_email":                cfg.ClientEmail,
		"client_id":                   cfg.ClientID,
		"auth_uri":                    cfg.AuthURI,
		"token_uri":                   cfg.TokenURI,
		"auth_provider_x509_cert_url": cfg.AuthProviderX509CertURL,
		"client_x509_cert_url":        cfg.ClientX509CertURL,
		// "universe_domain" not strictly needed by the library,
		// but if you want to keep it consistent, you can add it. Usually optional.
	}

	credJSON, err := json.Marshal(credMap)
	if err != nil {
		r.Log(robot.Fatal, "Unable to marshal GCP credentials from config: %v", err)
	}

	// Parse credentials
	creds, err := google.CredentialsFromJSON(ctx, credJSON, "https://www.googleapis.com/auth/datastore")
	if err != nil {
		r.Log(robot.Fatal, "Unable to parse Firestore credentials: %v", err)
	}

	// Create a Firestore client
	client, err := firestore.NewClient(ctx, cfg.ProjectID, option.WithCredentials(creds))
	if err != nil {
		r.Log(robot.Fatal, "Unable to create Firestore client: %v", err)
	}

	r.Log(robot.Au)

	// Return our fsBrain that implements SimpleBrain
	return &fsBrain{
		handler: r,
		config:  cfg,
		client:  client,
		ctx:     ctx,
		cancel:  cancel,
		colName: cfg.Collection,
	}
}

//------------------------------------------------------------------------------
//  D) Implement robot.SimpleBrain
//------------------------------------------------------------------------------

func (b *fsBrain) Store(key string, blob *[]byte) error {
	// Firestore: doc ID = key, field "content" = the []byte
	docRef := b.client.Collection(b.colName).Doc(key)

	_, err := docRef.Set(b.ctx, memoryDoc{Content: *blob})
	if err != nil {
		b.handler.Log(robot.Error, "Firestore Store error, key=%s: %v", key, err)
	}
	return err
}

func (b *fsBrain) Retrieve(key string) (blob *[]byte, exists bool, err error) {
	docRef := b.client.Collection(b.colName).Doc(key)
	snap, err := docRef.Get(b.ctx)
	if err != nil {
		// If doc not found
		if status.Code(err) == codes.NotFound {
			return nil, false, nil
		}
		b.handler.Log(robot.Error, "Firestore Retrieve error, key=%s: %v", key, err)
		return nil, false, err
	}
	// We found a document
	var md memoryDoc
	err = snap.DataTo(&md)
	if err != nil {
		b.handler.Log(robot.Error, "Firestore: failed to parse doc, key=%s: %v", key, err)
		return nil, false, err
	}
	return &md.Content, true, nil
}

func (b *fsBrain) Delete(key string) error {
	docRef := b.client.Collection(b.colName).Doc(key)
	_, err := docRef.Delete(b.ctx)
	if err != nil {
		// If it was not found, Firestore's Delete doesn't mind (it just returns no error)
		b.handler.Log(robot.Error, "Firestore Delete error, key=%s: %v", key, err)
	}
	return err
}

func (b *fsBrain) Shutdown() {
}

func (b *fsBrain) List() (keys []string, err error) {
	keys = make([]string, 0, 10)

	// We'll do a simple collection scan for doc IDs. For Gopherbot scale, that's typically small.
	// If you have thousands of docs, you might need pagination or batched queries.
	docs, err := b.client.Collection(b.colName).Documents(b.ctx).GetAll()
	if err != nil {
		b.handler.Log(robot.Error, "Firestore List error: %v", err)
		return keys, err
	}

	for _, d := range docs {
		keys = append(keys, d.Ref.ID)
	}
	return keys, nil
}

// ------------------------------------------------------------------------------
//
//	E) Optional: If you want a Stop() or Close() method
//
// ------------------------------------------------------------------------------
//
// The SimpleBrain interface doesn't define Stop(), but if you want to properly
// close the Firestore client in a graceful shutdown, you can do something like:
func (b *fsBrain) Stop() error {
	// Cancel the context
	b.cancel()

	// Close the client
	return b.client.Close()
}

//------------------------------------------------------------------------------
//  F) Note: Registration in static.go
//------------------------------------------------------------------------------
//
// For your "static.go" (or wherever you register this plugin):
//
//  package firestorebrain
//
//  import "github.com/lnxjedi/gopherbot/v2/bot"
//
//  func init() {
//      bot.RegisterSimpleBrain("gcpfirestore", provider)
//  }
