package bot

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

const userValidationCodeTTL = 30 * time.Second

type userValidationRequest struct {
	UserName      string
	AdminUser     string
	AdminProtocol string
	Created       time.Time
}

var userValidationRequests = struct {
	sync.Mutex
	byCode map[string]userValidationRequest
}{
	byCode: make(map[string]userValidationRequest),
}

func shouldAcceptIncomingUser(listedUser, ignoreUnlisted, validatedUser bool) bool {
	if ignoreUnlisted {
		return listedUser && validatedUser
	}
	if listedUser && !validatedUser {
		return false
	}
	return true
}

func isUserValidationCodeMessage(inc *robot.ConnectorMessage) (string, bool) {
	if inc == nil {
		return "", false
	}
	if !inc.DirectMessage && !inc.HiddenMessage {
		return "", false
	}
	code := strings.TrimSpace(inc.MessageText)
	if len(code) != 7 {
		return "", false
	}
	for _, ch := range code {
		if ch < '0' || ch > '9' {
			return "", false
		}
	}
	return code, true
}

func generateUserValidationCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%07d", n.Int64()), nil
}

func issueUserValidationRequest(userName, adminUser, adminProtocol string, now time.Time) (string, error) {
	request := userValidationRequest{
		UserName:      strings.ToLower(strings.TrimSpace(userName)),
		AdminUser:     strings.ToLower(strings.TrimSpace(adminUser)),
		AdminProtocol: normalizeProtocolName(adminProtocol),
		Created:       now,
	}
	if request.UserName == "" || request.AdminUser == "" || request.AdminProtocol == "" {
		return "", fmt.Errorf("invalid user validation request")
	}

	userValidationRequests.Lock()
	defer userValidationRequests.Unlock()

	for attempts := 0; attempts < 10; attempts++ {
		code, err := generateUserValidationCode()
		if err != nil {
			return "", err
		}
		if _, exists := userValidationRequests.byCode[code]; exists {
			continue
		}
		userValidationRequests.byCode[code] = request
		return code, nil
	}
	return "", fmt.Errorf("unable to allocate unique validation code")
}

func consumeUserValidationRequest(code string, now time.Time) (userValidationRequest, bool) {
	userValidationRequests.Lock()
	defer userValidationRequests.Unlock()

	request, ok := userValidationRequests.byCode[code]
	if !ok {
		return userValidationRequest{}, false
	}
	delete(userValidationRequests.byCode, code)
	if now.Sub(request.Created) > userValidationCodeTTL {
		return userValidationRequest{}, false
	}
	return request, true
}

func expireUserValidationRequests(now time.Time) {
	userValidationRequests.Lock()
	for code, request := range userValidationRequests.byCode {
		if now.Sub(request.Created) > userValidationCodeTTL {
			delete(userValidationRequests.byCode, code)
		}
	}
	userValidationRequests.Unlock()
}

func consumeIncomingUserValidationCode(inc *robot.ConnectorMessage) bool {
	code, ok := isUserValidationCodeMessage(inc)
	if !ok {
		return false
	}
	request, ok := consumeUserValidationRequest(code, time.Now())
	if !ok {
		return false
	}

	protocol := normalizeProtocolName(inc.Protocol)
	if protocol == "" {
		protocol = "unknown"
	}
	userID := strings.TrimSpace(inc.UserID)
	if userID == "" {
		userID = "(unknown)"
	}
	msg := fmt.Sprintf("User validation received: %s user '%s' has internal ID '%s'", protocol, request.UserName, userID)
	ret := interfaces.SendProtocolUserMessage(request.AdminUser, msg, robot.BasicMarkdown, &robot.ConnectorMessage{Protocol: request.AdminProtocol})
	if ret != robot.Ok {
		Log(robot.Warn, "User validation notification failed for admin '%s' on protocol '%s': %s", request.AdminUser, request.AdminProtocol, ret)
	}
	return true
}
