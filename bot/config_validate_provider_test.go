package bot

import (
	"strings"
	"testing"
)

func TestValidateYAMLBrainConfigFile(t *testing.T) {
	yml := []byte("BrainConfig:\n  AnyProviderKey: value\n")
	if err := validate_yaml("conf/brains/file.yaml", yml); err != nil {
		t.Fatalf("validate_yaml() rejected BrainConfig provider file: %v", err)
	}
}

func TestValidateYAMLBrainConfigFileRejectsUnknownTopLevel(t *testing.T) {
	yml := []byte("BrainConfig:\n  Foo: bar\nExtra: nope\n")
	err := validate_yaml("conf/brains/file.yaml", yml)
	if err == nil {
		t.Fatalf("validate_yaml() accepted unknown top-level key in brains config")
	}
	if !strings.Contains(err.Error(), "Extra") {
		t.Fatalf("validate_yaml() error %q did not reference unknown key", err)
	}
}

func TestValidateYAMLHistoryConfigFile(t *testing.T) {
	yml := []byte("HistoryConfig:\n  BufferSize: 16384\n")
	if err := validate_yaml("conf/history/mem.yaml", yml); err != nil {
		t.Fatalf("validate_yaml() rejected HistoryConfig provider file: %v", err)
	}
}

func TestValidateYAMLHistoryConfigFileRejectsUnknownTopLevel(t *testing.T) {
	yml := []byte("HistoryConfig:\n  BufferSize: 16384\nOther: bad\n")
	err := validate_yaml("conf/history/mem.yaml", yml)
	if err == nil {
		t.Fatalf("validate_yaml() accepted unknown top-level key in history config")
	}
	if !strings.Contains(err.Error(), "Other") {
		t.Fatalf("validate_yaml() error %q did not reference unknown key", err)
	}
}
