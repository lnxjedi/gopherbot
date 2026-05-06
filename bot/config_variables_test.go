package bot

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetConfigVariableTestState(t *testing.T) {
	t.Helper()
	oldConfigPath := configPath
	oldInstallPath := installPath
	oldVariables := activeConfigVariables.values
	oldInitialized := cryptKey.initialized
	oldInitializing := cryptKey.initializing
	oldKey := append([]byte(nil), cryptKey.key...)

	cryptKey.Lock()
	cryptKey.key = []byte("0123456789abcdef0123456789abcdef")
	cryptKey.initialized = true
	cryptKey.Unlock()
	setActiveConfigVariables(newConfigVariableSet())

	t.Cleanup(func() {
		configPath = oldConfigPath
		installPath = oldInstallPath
		setActiveConfigVariables(oldVariables)
		cryptKey.Lock()
		cryptKey.key = oldKey
		cryptKey.initialized = oldInitialized
		cryptKey.initializing = oldInitializing
		cryptKey.Unlock()
	})
}

func encryptedConfigSecretForTest(t *testing.T, plaintext string) string {
	t.Helper()
	cryptKey.RLock()
	key := append([]byte(nil), cryptKey.key...)
	cryptKey.RUnlock()
	ct, err := encrypt([]byte(plaintext), key)
	if err != nil {
		t.Fatalf("encrypt test secret: %v", err)
	}
	return base64.StdEncoding.EncodeToString(ct)
}

func TestLoadConfigVariablesCustomEnvironmentOverridesCommon(t *testing.T) {
	resetConfigVariableTestState(t)
	t.Setenv("GOPHER_ENVIRONMENT", "development")
	configPath = t.TempDir()
	installPath = t.TempDir()

	varDir := filepath.Join(configPath, "conf", "variables")
	if err := os.MkdirAll(varDir, 0700); err != nil {
		t.Fatalf("MkdirAll variables: %v", err)
	}
	commonSecret := encryptedConfigSecretForTest(t, "common-secret")
	envSecret := encryptedConfigSecretForTest(t, "env-secret")
	deletedSecret := encryptedConfigSecretForTest(t, "deleted-secret")
	if err := os.WriteFile(filepath.Join(varDir, "common.yaml"), []byte(`
Secrets:
  API_TOKEN: "`+commonSecret+`"
  DELETE_ME: "`+deletedSecret+`"
Variables:
  OUTPUT_CHANNEL: "common-jobs"
  DROP_ME: "drop"
`), 0600); err != nil {
		t.Fatalf("write common variables: %v", err)
	}
	if err := os.WriteFile(filepath.Join(varDir, "development.yaml"), []byte(`
Secrets:
  API_TOKEN: "`+envSecret+`"
  DELETE_ME: null
Variables:
  OUTPUT_CHANNEL: "dev-jobs"
  DROP_ME: null
`), 0600); err != nil {
		t.Fatalf("write environment variables: %v", err)
	}

	values, err := loadConfigVariables()
	if err != nil {
		t.Fatalf("loadConfigVariables(): %v", err)
	}
	if got := values.Secrets["API_TOKEN"]; got != envSecret {
		t.Fatalf("API_TOKEN = %q, want env secret", got)
	}
	if _, ok := values.Secrets["DELETE_ME"]; ok {
		t.Fatal("DELETE_ME secret survived null environment override")
	}
	if got := values.Variables["OUTPUT_CHANNEL"]; got != "dev-jobs" {
		t.Fatalf("OUTPUT_CHANNEL = %q", got)
	}
	if _, ok := values.Variables["DROP_ME"]; ok {
		t.Fatal("DROP_ME variable survived null environment override")
	}
}

func TestConfigVariablesIgnoreInstalledVariables(t *testing.T) {
	resetConfigVariableTestState(t)
	t.Setenv("GOPHER_ENVIRONMENT", "development")
	configPath = t.TempDir()
	installPath = t.TempDir()

	installedVarDir := filepath.Join(installPath, "conf", "variables")
	if err := os.MkdirAll(installedVarDir, 0700); err != nil {
		t.Fatalf("MkdirAll installed variables: %v", err)
	}
	if err := os.WriteFile(filepath.Join(installedVarDir, "development.yaml"), []byte(`
Variables:
  INSTALLED_ONLY: "must-not-load"
`), 0600); err != nil {
		t.Fatalf("write installed variables: %v", err)
	}

	values, err := loadConfigVariables()
	if err != nil {
		t.Fatalf("loadConfigVariables(): %v", err)
	}
	if _, ok := values.Variables["INSTALLED_ONLY"]; ok {
		t.Fatal("loaded variable from installed conf/variables")
	}
}

func TestSecretAndVariableTemplateFunctions(t *testing.T) {
	resetConfigVariableTestState(t)
	t.Setenv("GOPHER_ENVIRONMENT", "development")
	configPath = t.TempDir()
	varDir := filepath.Join(configPath, "conf", "variables")
	if err := os.MkdirAll(varDir, 0700); err != nil {
		t.Fatalf("MkdirAll variables: %v", err)
	}
	if err := os.WriteFile(filepath.Join(varDir, "development.yaml"), []byte(`
Secrets:
  API_TOKEN: "`+encryptedConfigSecretForTest(t, "secret-value")+`"
Variables:
  OUTPUT_CHANNEL: "dev-jobs"
`), 0600); err != nil {
		t.Fatalf("write variables: %v", err)
	}
	values, err := loadConfigVariables()
	if err != nil {
		t.Fatalf("loadConfigVariables(): %v", err)
	}
	setActiveConfigVariables(values)

	out, err := expand("conf", true, []byte(`token={{ secret "API_TOKEN" }} channel={{ variable "OUTPUT_CHANNEL" }}`))
	if err != nil {
		t.Fatalf("expand(): %v", err)
	}
	if got := string(out); got != "token=secret-value channel=dev-jobs" {
		t.Fatalf("expanded = %q", got)
	}
}

func TestDecryptTemplateFunctionFailsWithMigrationHint(t *testing.T) {
	resetConfigVariableTestState(t)
	_, err := expand("conf", true, []byte(`value={{ decrypt "abc" }}`))
	if err == nil {
		t.Fatal("expand with decrypt succeeded")
	}
	if got := err.Error(); !strings.Contains(got, `template function "decrypt" was removed in v3`) || !strings.Contains(got, `{{ secret "NAME" }}`) {
		t.Fatalf("decrypt error missing migration hint: %v", err)
	}
}

func TestConfigVariablesRejectInvalidEnvironmentName(t *testing.T) {
	resetConfigVariableTestState(t)
	t.Setenv("GOPHER_ENVIRONMENT", "../development")
	configPath = t.TempDir()

	if _, err := loadConfigVariables(); err == nil {
		t.Fatal("loadConfigVariables succeeded with invalid environment")
	}
}

func TestCliGenKeyWritesEnvironmentSpecificDecryptableKey(t *testing.T) {
	resetConfigVariableTestState(t)
	configPath = t.TempDir()
	t.Setenv("GOPHER_ENCRYPTION_KEY", "12345678901234567890123456789012")

	if err := cliGenKey("development", true, false); err != nil {
		t.Fatalf("cliGenKey(): %v", err)
	}
	path := filepath.Join(configPath, encryptedKeyFile+".development")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading generated key: %v", err)
	}
	ct, err := base64.StdEncoding.DecodeString(string(payload))
	if err != nil {
		t.Fatalf("generated key is not base64: %v", err)
	}
	key, err := decrypt(ct, []byte("12345678901234567890123456789012"))
	if err != nil {
		t.Fatalf("decrypting generated key: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("generated data key length = %d, want 32", len(key))
	}
}

func TestCliGenKeyRefusesExistingKeyWithoutForce(t *testing.T) {
	resetConfigVariableTestState(t)
	configPath = t.TempDir()
	t.Setenv("GOPHER_ENCRYPTION_KEY", "12345678901234567890123456789012")
	path := filepath.Join(configPath, encryptedKeyFile)
	if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
		t.Fatalf("write existing key: %v", err)
	}

	if err := cliGenKey("production", true, false); err == nil {
		t.Fatal("cliGenKey replaced existing key without force")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read existing key: %v", err)
	}
	if string(got) != "existing" {
		t.Fatalf("existing key was modified: %q", string(got))
	}
}
