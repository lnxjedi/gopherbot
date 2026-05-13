package gcloud

import (
	"context"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/pubsub"
	"github.com/lnxjedi/gopherbot/robot"
	gcloudinternal "github.com/lnxjedi/gopherbot/v2/internal/gcloud"
)

const (
	defaultCredentialsEncryptedFile = "gopherbot-key.json.enc"
	defaultSubscriptionID           = "job-triggers-pull"
)

type config struct {
	ProjectID                string
	SubscriptionID           string
	CredentialsEncryptedFile string
	MaxOutstandingMessages   int
	NumGoroutines            int
}

type queueProvider struct {
	robot.QueueHandler
	projectID      string
	subscriptionID string
	client         *pubsub.Client
	subscription   *pubsub.Subscription
}

func Initialize(handler robot.QueueHandler, _ *log.Logger) (robot.InitializedQueueProvider, error) {
	var c config
	if err := handler.GetQueueConfig(&c); err != nil {
		return robot.InitializedQueueProvider{}, fmt.Errorf("retrieve gcloud queue configuration: %w", err)
	}

	credentialsPath := strings.TrimSpace(c.CredentialsEncryptedFile)
	if credentialsPath == "" {
		credentialsPath = defaultCredentialsEncryptedFile
	}
	creds, err := gcloudinternal.LoadServiceAccountCredentials(handler.ReadEncryptedFile, credentialsPath)
	if err != nil {
		return robot.InitializedQueueProvider{}, fmt.Errorf("load Google credentials: %w", err)
	}
	projectID, err := gcloudinternal.ResolveProjectID(c.ProjectID, creds)
	if err != nil {
		return robot.InitializedQueueProvider{}, fmt.Errorf("determine Google project ID: %w", err)
	}
	subscriptionID := normalizeSubscriptionID(c.SubscriptionID)
	if subscriptionID == "" {
		subscriptionID = defaultSubscriptionID
	}

	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID, gcloudinternal.ServiceAccountClientOptions(creds)...)
	if err != nil {
		return robot.InitializedQueueProvider{}, fmt.Errorf("create Google Pub/Sub client: %w", err)
	}
	subscription := client.Subscription(subscriptionID)
	if c.NumGoroutines > 0 {
		subscription.ReceiveSettings.NumGoroutines = c.NumGoroutines
	} else {
		subscription.ReceiveSettings.NumGoroutines = 1
	}
	if c.MaxOutstandingMessages > 0 {
		subscription.ReceiveSettings.MaxOutstandingMessages = c.MaxOutstandingMessages
	} else {
		subscription.ReceiveSettings.MaxOutstandingMessages = 1
	}

	qp := &queueProvider{
		QueueHandler:   handler,
		projectID:      projectID,
		subscriptionID: subscriptionID,
		client:         client,
		subscription:   subscription,
	}
	return robot.InitializedQueueProvider{Provider: qp}, nil
}

func normalizeSubscriptionID(in string) string {
	in = strings.TrimSpace(in)
	if in == "" {
		return ""
	}
	if idx := strings.LastIndex(in, "/subscriptions/"); idx >= 0 {
		return strings.TrimSpace(in[idx+len("/subscriptions/"):])
	}
	if idx := strings.LastIndex(in, "subscriptions/"); idx >= 0 {
		return strings.TrimSpace(in[idx+len("subscriptions/"):])
	}
	return in
}

func (q *queueProvider) Run(stop <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		if err := q.client.Close(); err != nil {
			q.Log(robot.Warn, "Closing Google Pub/Sub client: %v", err)
		}
	}()

	stopWaitDone := make(chan struct{})
	defer close(stopWaitDone)
	go func() {
		select {
		case <-stop:
			cancel()
		case <-stopWaitDone:
		}
	}()

	q.Log(robot.Info, "Google Pub/Sub queue provider receiving from projects/%s/subscriptions/%s", q.projectID, q.subscriptionID)
	err := q.subscription.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		disposition := q.HandleQueueMessage(robot.QueueMessage{
			ID:         msg.ID,
			Body:       msg.Data,
			Attributes: msg.Attributes,
		})
		switch disposition {
		case robot.QueueRetry:
			msg.Nack()
		default:
			msg.Ack()
		}
	})
	if err != nil && ctx.Err() == nil {
		q.Log(robot.Error, "Google Pub/Sub queue receive loop exited with error: %v", err)
	}
}
