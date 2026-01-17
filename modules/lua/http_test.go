package lua

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/stretchr/testify/assert"
	"github.com/yuin/gopher-lua"
)

// mockLogger is a simple logger for testing
type mockLogger struct {
	messages []string
}

func (m *mockLogger) Log(level robot.LogLevel, msg string, v ...interface{}) bool {
	var levelStr string
	switch level {
	case robot.Trace:
		levelStr = "TRACE"
	case robot.Debug:
		levelStr = "DEBUG"
	case robot.Info:
		levelStr = "INFO"
	case robot.Audit:
		levelStr = "AUDIT"
	case robot.Warn:
		levelStr = "WARN"
	case robot.Error:
		levelStr = "ERROR"
	case robot.Fatal:
		levelStr = "FATAL"
	}
	m.messages = append(m.messages, fmt.Sprintf(levelStr+": "+msg, v...))
	return true
}

// mockRobot is a mock robot for testing
type mockRobot struct {
	robot.Robot
	logger *mockLogger
}

func (m *mockRobot) Log(level robot.LogLevel, msg string, v ...interface{}) bool {
	return m.logger.Log(level, msg, v...)
}

func TestHttpClient(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if r.Header.Get("Content-Type") != "application/json" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintln(w, `{"status":"created"}`)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "Hello, client")
		}
	}))
	defer server.Close()

	// Get the current working directory to build the absolute path to the lib directory
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	libPath := filepath.Join(wd, "..", "..", "lib")
	luaPath := fmt.Sprintf("%s/?.lua", libPath)

	// Lua script to test the HTTP client
	luaScript := `
		package.path = package.path .. ';` + luaPath + `'
		local bot = require("gopherbot_v1")
		local r = bot.Robot:new()
		local client = http.new("` + server.URL + `", {})

		client:get("/test", nil, function(resp, err)
			if err then
				r:Log(bot.log.Error, "GET request failed: " .. err)
				return
			end
			r:Log(bot.log.Info, "GET response status: " .. resp.statusCode)
			r:Log(bot.log.Info, "GET response body: " .. resp.body)
		end)

		local postClient = http.new("` + server.URL + `", {
			headers = { ["Content-Type"] = "application/json" }
		})
		postClient:post("/test", "{\"data\":\"value\"}", nil, function(resp, err)
			if err then
				r:Log(bot.log.Error, "POST request failed: " .. err)
				return
			end
			r:Log(bot.log.Info, "POST response status: " .. resp.statusCode)
			r:Log(bot.log.Info, "POST response body: " .. resp.body)
		end)
	`

	// Create a new Lua state
	L := lua.NewState()
	defer L.Close()

	// Mock the robot logger
	logger := &mockLogger{}

	// Create a mock robot
	mockBot := &mockRobot{logger: logger}

	// Create a luaContext
	lctx := &luaContext{
		Logger: logger,
		L:      L,
	}

	// Register the bot metatable and methods
	registerBotMetatableIfNeeded(L)
	lctx.RegisterMessageMethods(L)
	lctx.RegisterUtilMethods(L)

	// Add the http handler
	addHttpHandler(L)

	// Create a mock robot userdata
	robotUD := lctx.newLuaBot(L, mockBot)
	L.SetGlobal("GBOT", robotUD)

	// Run the script
	err = L.DoString(luaScript)
	assert.NoError(t, err)

	// Verify the log output
	assert.Contains(t, strings.Join(logger.messages, "\n"), "INFO: GET response status: 200")
	assert.Contains(t, strings.Join(logger.messages, "\n"), "INFO: GET response body: Hello, client")
	assert.Contains(t, strings.Join(logger.messages, "\n"), "INFO: POST response status: 201")
	assert.Contains(t, strings.Join(logger.messages, "\n"), "INFO: POST response body: {\"status\":\"created\"}")
}

