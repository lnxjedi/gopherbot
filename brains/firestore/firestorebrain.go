package firestorebrain

import (
	"context"
	"sort"
	"strings"
	"time"

	firestoreapi "cloud.google.com/go/firestore"
	"github.com/lnxjedi/gopherbot/robot"
	"github.com/lnxjedi/gopherbot/v2/internal/gcloud"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type brainConfig struct {
	ProjectID                string
	DatabaseID               string
	Collection               string
	CredentialsEncryptedFile string
	OperationTimeoutSeconds  int
}

type storedMemory struct {
	Content []byte `firestore:"content"`
}

type firestoreBrain struct {
	cfg    brainConfig
	client *firestoreapi.Client
}

func defaultedConfig(cfg brainConfig) brainConfig {
	if strings.TrimSpace(cfg.DatabaseID) == "" {
		cfg.DatabaseID = "(default)"
	}
	if strings.TrimSpace(cfg.Collection) == "" {
		cfg.Collection = "gopherbot-brain"
	}
	if cfg.OperationTimeoutSeconds <= 0 {
		cfg.OperationTimeoutSeconds = 15
	}
	return cfg
}

func (b *firestoreBrain) timeoutContext() (context.Context, context.CancelFunc) {
	timeout := time.Duration(b.cfg.OperationTimeoutSeconds) * time.Second
	return context.WithTimeout(context.Background(), timeout)
}

func (b *firestoreBrain) Store(key string, blob *[]byte) error {
	ctx, cancel := b.timeoutContext()
	defer cancel()

	_, err := b.client.Collection(b.cfg.Collection).Doc(key).Set(ctx, storedMemory{
		Content: *blob,
	})
	return err
}

func (b *firestoreBrain) Retrieve(key string) (datum *[]byte, exists bool, err error) {
	ctx, cancel := b.timeoutContext()
	defer cancel()

	doc, err := b.client.Collection(b.cfg.Collection).Doc(key).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	if doc == nil || !doc.Exists() {
		return nil, false, nil
	}

	var stored storedMemory
	if err := doc.DataTo(&stored); err != nil {
		return nil, false, err
	}
	return &stored.Content, true, nil
}

func (b *firestoreBrain) Delete(key string) error {
	ctx, cancel := b.timeoutContext()
	defer cancel()

	_, err := b.client.Collection(b.cfg.Collection).Doc(key).Delete(ctx)
	return err
}

func (b *firestoreBrain) List() ([]string, error) {
	ctx, cancel := b.timeoutContext()
	defer cancel()

	refs, err := b.client.Collection(b.cfg.Collection).DocumentRefs(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref == nil {
			continue
		}
		keys = append(keys, ref.ID)
	}
	sort.Strings(keys)
	return keys, nil
}

func (b *firestoreBrain) Shutdown() {
	if b.client != nil {
		_ = b.client.Close()
	}
}

func provider(r robot.Handler) robot.SimpleBrain {
	var cfg brainConfig
	if err := r.GetBrainConfig(&cfg); err != nil {
		r.Log(robot.Fatal, "Unable to retrieve Firestore brain configuration: %v", err)
	}
	cfg = defaultedConfig(cfg)

	var clientOptions []option.ClientOption
	if strings.TrimSpace(cfg.CredentialsEncryptedFile) != "" {
		creds, err := gcloud.LoadServiceAccountCredentials(r.ReadEncryptedFile, cfg.CredentialsEncryptedFile)
		if err != nil {
			r.Log(robot.Fatal, "Loading encrypted Google credentials for Firestore brain: %v", err)
		}
		projectID, err := gcloud.ResolveProjectID(cfg.ProjectID, creds)
		if err != nil {
			r.Log(robot.Fatal, "Resolving Google project ID for Firestore brain: %v", err)
		}
		cfg.ProjectID = projectID
		clientOptions = gcloud.ServiceAccountClientOptions(creds)
	} else if projectID, err := gcloud.ResolveProjectID(cfg.ProjectID, nil); err == nil {
		cfg.ProjectID = projectID
	}

	if strings.TrimSpace(cfg.ProjectID) == "" {
		r.Log(robot.Fatal, "Firestore brain requires ProjectID or CredentialsEncryptedFile with project_id")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.OperationTimeoutSeconds)*time.Second)
	defer cancel()

	var (
		client *firestoreapi.Client
		err    error
	)
	if cfg.DatabaseID == "(default)" {
		client, err = firestoreapi.NewClient(ctx, cfg.ProjectID, clientOptions...)
	} else {
		client, err = firestoreapi.NewClientWithDatabase(ctx, cfg.ProjectID, cfg.DatabaseID, clientOptions...)
	}
	if err != nil {
		r.Log(robot.Fatal, "Creating Firestore client for project '%s': %v", cfg.ProjectID, err)
	}

	pingDoc := client.Collection(cfg.Collection).Doc("gopherbot-ping")
	if _, err := pingDoc.Get(ctx); err != nil && status.Code(err) != codes.NotFound {
		_ = client.Close()
		r.Log(robot.Fatal, "Validating Firestore brain collection '%s': %v", cfg.Collection, err)
	}

	r.Log(robot.Info, "Initialized Firestore brain for project '%s', database '%s', collection '%s'", cfg.ProjectID, cfg.DatabaseID, cfg.Collection)
	return &firestoreBrain{
		cfg:    cfg,
		client: client,
	}
}
