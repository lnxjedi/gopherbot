package googlechat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const workspaceEventsBaseURL = "https://workspaceevents.googleapis.com/v1"

type workspaceEventsClient struct {
	httpClient *http.Client
	baseURL    string
}

type workspaceSubscription struct {
	Name                    string                         `json:"name,omitempty"`
	TargetResource          string                         `json:"targetResource,omitempty"`
	EventTypes              []string                       `json:"eventTypes,omitempty"`
	NotificationEndpoint    *workspaceNotificationEndpoint `json:"notificationEndpoint,omitempty"`
	PayloadOptions          *workspacePayloadOptions       `json:"payloadOptions,omitempty"`
	State                   string                         `json:"state,omitempty"`
	SuspensionReason        string                         `json:"suspensionReason,omitempty"`
	Etag                    string                         `json:"etag,omitempty"`
	ExpireTime              string                         `json:"expireTime,omitempty"`
	ServiceAccountAuthority string                         `json:"serviceAccountAuthority,omitempty"`
	Ttl                     string                         `json:"ttl,omitempty"`
}

func (s *workspaceSubscription) UnmarshalJSON(data []byte) error {
	type alias workspaceSubscription
	var raw struct {
		alias
		TargetResourceSnake          string                         `json:"target_resource,omitempty"`
		EventTypesSnake              []string                       `json:"event_types,omitempty"`
		NotificationEndpointSnake    *workspaceNotificationEndpoint `json:"notification_endpoint,omitempty"`
		PayloadOptionsSnake          *workspacePayloadOptions       `json:"payload_options,omitempty"`
		SuspensionReasonSnake        string                         `json:"suspension_reason,omitempty"`
		ExpireTimeSnake              string                         `json:"expire_time,omitempty"`
		ServiceAccountAuthoritySnake string                         `json:"service_account_authority,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*s = workspaceSubscription(raw.alias)
	if s.TargetResource == "" {
		s.TargetResource = raw.TargetResourceSnake
	}
	if len(s.EventTypes) == 0 {
		s.EventTypes = raw.EventTypesSnake
	}
	if s.NotificationEndpoint == nil {
		s.NotificationEndpoint = raw.NotificationEndpointSnake
	}
	if s.PayloadOptions == nil {
		s.PayloadOptions = raw.PayloadOptionsSnake
	}
	if s.SuspensionReason == "" {
		s.SuspensionReason = raw.SuspensionReasonSnake
	}
	if s.ExpireTime == "" {
		s.ExpireTime = raw.ExpireTimeSnake
	}
	if s.ServiceAccountAuthority == "" {
		s.ServiceAccountAuthority = raw.ServiceAccountAuthoritySnake
	}
	return nil
}

type workspaceNotificationEndpoint struct {
	PubsubTopic string `json:"pubsubTopic,omitempty"`
}

func (e *workspaceNotificationEndpoint) UnmarshalJSON(data []byte) error {
	var raw struct {
		PubsubTopic      string `json:"pubsubTopic,omitempty"`
		PubsubTopicSnake string `json:"pubsub_topic,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	e.PubsubTopic = raw.PubsubTopic
	if e.PubsubTopic == "" {
		e.PubsubTopic = raw.PubsubTopicSnake
	}
	return nil
}

type workspacePayloadOptions struct {
	IncludeResource bool   `json:"includeResource,omitempty"`
	FieldMask       string `json:"fieldMask,omitempty"`
}

func (o *workspacePayloadOptions) UnmarshalJSON(data []byte) error {
	var raw struct {
		IncludeResource      bool   `json:"includeResource,omitempty"`
		IncludeResourceSnake bool   `json:"include_resource,omitempty"`
		FieldMask            string `json:"fieldMask,omitempty"`
		FieldMaskSnake       string `json:"field_mask,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	o.IncludeResource = raw.IncludeResource || raw.IncludeResourceSnake
	o.FieldMask = raw.FieldMask
	if o.FieldMask == "" {
		o.FieldMask = raw.FieldMaskSnake
	}
	return nil
}

type workspaceListSubscriptionsResponse struct {
	Subscriptions []workspaceSubscription `json:"subscriptions,omitempty"`
	NextPageToken string                  `json:"nextPageToken,omitempty"`
}

type workspaceOperation struct {
	Name     string                `json:"name,omitempty"`
	Done     bool                  `json:"done,omitempty"`
	Response json.RawMessage       `json:"response,omitempty"`
	Error    *workspaceStatusError `json:"error,omitempty"`
}

type workspaceStatusError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type workspaceAPIError struct {
	Method     string
	Path       string
	StatusCode int
	Status     string
	Message    string
	Reason     string
	Domain     string
	Body       string
}

func (e *workspaceAPIError) Error() string {
	if e == nil {
		return ""
	}
	parts := []string{
		fmt.Sprintf("workspace events API %s %s failed: HTTP %d", e.Method, e.Path, e.StatusCode),
	}
	if e.Status != "" {
		parts = append(parts, "status="+e.Status)
	}
	if e.Reason != "" {
		parts = append(parts, "reason="+e.Reason)
	}
	if e.Domain != "" {
		parts = append(parts, "domain="+e.Domain)
	}
	if e.Message != "" {
		parts = append(parts, fmt.Sprintf("message=%q", e.Message))
	} else if e.Body != "" {
		parts = append(parts, fmt.Sprintf("body=%q", e.Body))
	}
	return strings.Join(parts, " ")
}

func newWorkspaceEventsClient(httpClient *http.Client) *workspaceEventsClient {
	if httpClient == nil {
		return nil
	}
	return &workspaceEventsClient{
		httpClient: httpClient,
		baseURL:    workspaceEventsBaseURL,
	}
}

func (c *workspaceEventsClient) listSubscriptions(ctx context.Context, filter string) ([]workspaceSubscription, error) {
	if c == nil {
		return nil, fmt.Errorf("workspace events client is not configured")
	}
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return nil, fmt.Errorf("workspace events list filter is required")
	}

	var all []workspaceSubscription
	pageToken := ""
	for {
		query := url.Values{}
		query.Set("filter", filter)
		query.Set("pageSize", "100")
		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}

		var resp workspaceListSubscriptionsResponse
		if err := c.doJSON(ctx, http.MethodGet, "/subscriptions", query, nil, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Subscriptions...)
		if strings.TrimSpace(resp.NextPageToken) == "" {
			return all, nil
		}
		pageToken = resp.NextPageToken
	}
}

func (c *workspaceEventsClient) createSubscription(ctx context.Context, subscription *workspaceSubscription) (*workspaceSubscription, error) {
	return c.runSubscriptionOperation(ctx, http.MethodPost, "/subscriptions", nil, subscription)
}

func (c *workspaceEventsClient) updateSubscription(ctx context.Context, subscription *workspaceSubscription, updateMask string) (*workspaceSubscription, error) {
	if subscription == nil || strings.TrimSpace(subscription.Name) == "" {
		return nil, fmt.Errorf("subscription name is required")
	}
	query := url.Values{}
	if strings.TrimSpace(updateMask) != "" {
		query.Set("updateMask", updateMask)
	}
	return c.runSubscriptionOperation(ctx, http.MethodPatch, "/"+subscription.Name, query, subscription)
}

func (c *workspaceEventsClient) reactivateSubscription(ctx context.Context, name string) (*workspaceSubscription, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("subscription name is required")
	}
	return c.runSubscriptionOperation(ctx, http.MethodPost, "/"+name+":reactivate", nil, struct{}{})
}

func (c *workspaceEventsClient) deleteSubscription(ctx context.Context, name string, allowMissing bool) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("subscription name is required")
	}
	query := url.Values{}
	if allowMissing {
		query.Set("allowMissing", "true")
	}
	return c.doJSON(ctx, http.MethodDelete, "/"+name, query, nil, nil)
}

func (c *workspaceEventsClient) getOperation(ctx context.Context, name string) (*workspaceOperation, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("operation name is required")
	}
	var op workspaceOperation
	if err := c.doJSON(ctx, http.MethodGet, "/"+name, nil, nil, &op); err != nil {
		return nil, err
	}
	return &op, nil
}

func (c *workspaceEventsClient) runSubscriptionOperation(ctx context.Context, method, path string, query url.Values, body interface{}) (*workspaceSubscription, error) {
	if c == nil {
		return nil, fmt.Errorf("workspace events client is not configured")
	}
	var op workspaceOperation
	if err := c.doJSON(ctx, method, path, query, body, &op); err != nil {
		return nil, err
	}
	return c.waitOperationForSubscription(ctx, &op)
}

func (c *workspaceEventsClient) waitOperationForSubscription(ctx context.Context, op *workspaceOperation) (*workspaceSubscription, error) {
	if op == nil {
		return nil, fmt.Errorf("workspace events operation response is missing")
	}
	if !op.Done && strings.TrimSpace(op.Name) != "" {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for !op.Done {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-ticker.C:
				next, err := c.getOperation(ctx, op.Name)
				if err != nil {
					return nil, err
				}
				op = next
			}
		}
	}
	if op.Error != nil && op.Error.Message != "" {
		return nil, fmt.Errorf("workspace events operation failed (%d): %s", op.Error.Code, op.Error.Message)
	}
	if len(op.Response) == 0 {
		return nil, fmt.Errorf("workspace events operation completed without a subscription response")
	}
	var subscription workspaceSubscription
	if err := json.Unmarshal(op.Response, &subscription); err != nil {
		return nil, fmt.Errorf("parsing workspace events operation response: %w", err)
	}
	return &subscription, nil
}

func (c *workspaceEventsClient) doJSON(ctx context.Context, method, path string, query url.Values, body interface{}, out interface{}) error {
	if c == nil || c.httpClient == nil {
		return fmt.Errorf("workspace events HTTP client is not configured")
	}
	fullURL := strings.TrimRight(c.baseURL, "/") + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding workspace events request body: %w", err)
		}
		payload = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, payload)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseWorkspaceAPIError(method, path, resp.StatusCode, respBody)
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("parsing workspace events API response: %w", err)
	}
	return nil
}

func parseWorkspaceAPIError(method, path string, statusCode int, body []byte) error {
	apiErr := &workspaceAPIError{
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		Body:       compactHTTPBody(body),
	}
	var payload struct {
		Error struct {
			Code    int    `json:"code,omitempty"`
			Message string `json:"message,omitempty"`
			Status  string `json:"status,omitempty"`
			Details []struct {
				Type   string            `json:"@type,omitempty"`
				Reason string            `json:"reason,omitempty"`
				Domain string            `json:"domain,omitempty"`
				Meta   map[string]string `json:"metadata,omitempty"`
			} `json:"details,omitempty"`
		} `json:"error,omitempty"`
	}
	if len(body) > 0 && json.Unmarshal(body, &payload) == nil {
		apiErr.Message = strings.TrimSpace(payload.Error.Message)
		apiErr.Status = strings.TrimSpace(payload.Error.Status)
		for _, detail := range payload.Error.Details {
			if strings.TrimSpace(detail.Reason) == "" && strings.TrimSpace(detail.Domain) == "" {
				continue
			}
			apiErr.Reason = strings.TrimSpace(detail.Reason)
			apiErr.Domain = strings.TrimSpace(detail.Domain)
			break
		}
	}
	return apiErr
}

func compactHTTPError(statusCode int, body []byte) string {
	msg := compactHTTPBody(body)
	if msg == "" {
		return fmt.Sprintf("HTTP %d", statusCode)
	}
	return fmt.Sprintf("HTTP %d: %s", statusCode, msg)
}

func compactHTTPBody(body []byte) string {
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		return ""
	}
	if len(msg) > 400 {
		msg = msg[:400]
	}
	return msg
}
