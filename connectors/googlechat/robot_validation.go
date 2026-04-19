package googlechat

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	chatapi "google.golang.org/api/chat/v1"
)

var googleChatValidationCodeRe = regexp.MustCompile(`\b\d{7}\b`)

type robotValidationRequest struct {
	ResultCh chan robotValidationResult
	Created  time.Time
}

type robotValidationResult struct {
	BotID     string
	AckSpace  string
	AckThread string
}

func generateGoogleChatValidationCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%07d", n.Int64()), nil
}

// IssueRobotValidation issues a short-lived code for discovering the
// connector's numeric bot user ID from a live @mention event.
func (gc *googleChatConnector) IssueRobotValidation() (string, <-chan robotValidationResult, error) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	if gc.selfID != "" {
		return gc.selfID, nil, nil
	}
	if gc.robotValidation == nil {
		gc.robotValidation = make(map[string]robotValidationRequest)
	}
	now := time.Now()
	cutoff := now.Add(-robotValidationTTL)
	for code, request := range gc.robotValidation {
		if request.Created.Before(cutoff) {
			delete(gc.robotValidation, code)
		}
	}
	request := robotValidationRequest{
		ResultCh: make(chan robotValidationResult, 1),
		Created:  now,
	}
	for attempts := 0; attempts < 10; attempts++ {
		code, err := generateGoogleChatValidationCode()
		if err != nil {
			return "", nil, err
		}
		if _, exists := gc.robotValidation[code]; exists {
			continue
		}
		gc.robotValidation[code] = request
		return code, request.ResultCh, nil
	}
	return "", nil, fmt.Errorf("unable to allocate validation code")
}

func (gc *googleChatConnector) CurrentSelfID() string {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return gc.selfID
}

func (gc *googleChatConnector) consumeRobotValidationForEvent(event *chatEvent) bool {
	if gc == nil || event == nil || event.Message == nil {
		return false
	}
	code := extractGoogleChatValidationCode(event.Message.Text)
	if code == "" {
		return false
	}
	botID := gc.eventMentionedBotID(event.Message)
	if botID == "" || botID == "users/app" {
		return false
	}
	request, ok := gc.consumeRobotValidationCode(code)
	if !ok {
		return false
	}
	gc.learnSelfID(botID)

	space := event.Space
	if space == nil {
		space = event.Message.Space
	}
	spaceName := ""
	if space != nil {
		spaceName = strings.TrimSpace(space.Name)
	}
	threadID, _ := resolveThreadContext(event.Message, event.Thread)
	gc.fulfillRobotValidation(request, robotValidationResult{
		BotID:     botID,
		AckSpace:  spaceName,
		AckThread: threadID,
	})
	return true
}

func (gc *googleChatConnector) consumeRobotValidationForAPIMessage(message *chatapi.Message) bool {
	if gc == nil || message == nil {
		return false
	}
	code := extractGoogleChatValidationCode(message.Text)
	if code == "" {
		return false
	}
	botID := gc.apiMentionedBotID(message)
	if botID == "" || botID == "users/app" {
		return false
	}
	request, ok := gc.consumeRobotValidationCode(code)
	if !ok {
		return false
	}
	gc.learnSelfID(botID)

	spaceName := ""
	if message.Space != nil {
		spaceName = strings.TrimSpace(message.Space.Name)
	}
	threadID := ""
	if message.Thread != nil {
		threadID = strings.TrimSpace(message.Thread.Name)
	}
	gc.fulfillRobotValidation(request, robotValidationResult{
		BotID:     botID,
		AckSpace:  spaceName,
		AckThread: threadID,
	})
	return true
}

func extractGoogleChatValidationCode(text string) string {
	return strings.TrimSpace(googleChatValidationCodeRe.FindString(text))
}

func (gc *googleChatConnector) consumeRobotValidationCode(code string) (robotValidationRequest, bool) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	if gc.robotValidation == nil {
		return robotValidationRequest{}, false
	}
	now := time.Now()
	cutoff := now.Add(-robotValidationTTL)
	for pendingCode, request := range gc.robotValidation {
		if request.Created.Before(cutoff) {
			delete(gc.robotValidation, pendingCode)
		}
	}
	request, ok := gc.robotValidation[code]
	if !ok {
		return robotValidationRequest{}, false
	}
	delete(gc.robotValidation, code)
	return request, !request.Created.Before(cutoff)
}

func (gc *googleChatConnector) learnSelfID(botID string) {
	botID = normalizeUserResource(botID)
	if botID == "" || botID == "users/app" {
		return
	}
	gc.mu.Lock()
	gc.selfID = botID
	gc.mu.Unlock()
	gc.Handler.SetBotID(botID)
}

func (gc *googleChatConnector) CancelRobotValidation(code string) {
	if gc == nil || strings.TrimSpace(code) == "" {
		return
	}
	gc.mu.Lock()
	delete(gc.robotValidation, code)
	gc.mu.Unlock()
}

func (gc *googleChatConnector) fulfillRobotValidation(request robotValidationRequest, result robotValidationResult) {
	if gc == nil || request.ResultCh == nil {
		return
	}
	select {
	case request.ResultCh <- result:
	default:
		gc.Log(robot.Warn, "Dropping Google Chat robot validation result for bot ID %q; requester is no longer waiting", result.BotID)
	}
}
