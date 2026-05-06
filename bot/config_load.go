package bot

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"text/template"

	"github.com/lnxjedi/gopherbot/robot"
	"gopkg.in/yaml.v3"
)

const appendPrefix = "Append"

// merge map merges maps and concatenates slices; values in m(erge) override values
// in t(arget).
func mergemap(m, t map[string]interface{}) map[string]interface{} {
	for k, v := range m {
		appendArr := false
		if strings.HasPrefix(k, appendPrefix) {
			k = strings.TrimPrefix(k, appendPrefix)
			appendArr = true
		}
		if tv, ok := t[k]; ok {
			if reflect.TypeOf(v) == reflect.TypeOf(tv) {
				switch v.(type) {
				case map[string]interface{}:
					mv := v.(map[string]interface{})
					mtv := tv.(map[string]interface{})
					t[k] = mergemap(mv, mtv)
				case []interface{}:
					sv := v.([]interface{})
					if !appendArr {
						t[k] = sv
					} else {
						tva := tv.([]interface{})
						t[k] = append(tva, sv...)
					}
				default:
					t[k] = v
				}
			} else {
				// mis-matched types, use new value if non-nil
				if v != nil {
					t[k] = v
				} else {
					t[k] = tv
				}
			}
		} else {
			t[k] = v
		}
	}
	return t
}

// env is for the config file template FuncMap. It returns
// the given environment var if found. If required is set,
// an Error-level log event is generated for empty vars.
func env(envvar string) string {
	val := getEnv(envvar)
	if len(val) == 0 {
		Log(robot.Debug, "Empty environment variable returned for '%s' in template expansion", envvar)
	}
	return val
}

// defval is for the config file template FuncMap. If an empty string is piped in,
// the default value is returned.
func defval(d, i string) string {
	if len(i) == 0 {
		return d
	}
	return i
}

// setEnvVar allows the default robot.yaml to override environment variables
// seen when loading the custom robot.yaml.
func setEnvVar(k, v string) (err error) {
	err = setEnv(k, v)
	if err != nil {
		Log(robot.Error, "failed override setting '%s' to '%s': %v", k, v, err.Error())
	} else {
		Log(robot.Warn, "environment variable override while loading configuration, setting '%s=%s'", k, v)
	}
	return
}

type configVariableSet struct {
	Secrets   map[string]string
	Variables map[string]string
}

type configVariablesFile struct {
	Secrets   map[string]*string `yaml:"Secrets"`
	Variables map[string]*string `yaml:"Variables"`
}

var activeConfigVariables = struct {
	sync.RWMutex
	values *configVariableSet
}{
	values: &configVariableSet{
		Secrets:   map[string]string{},
		Variables: map[string]string{},
	},
}

func newConfigVariableSet() *configVariableSet {
	return &configVariableSet{
		Secrets:   make(map[string]string),
		Variables: make(map[string]string),
	}
}

func currentConfigTemplateEnvironment() string {
	env := strings.TrimSpace(currentDeployEnvironment())
	if env == "" {
		env = "production"
	}
	return env
}

func validateConfigTemplateEnvironment(env string) error {
	if env == "" || env == "." || env == ".." {
		return fmt.Errorf("invalid GOPHER_ENVIRONMENT %q", env)
	}
	if strings.ContainsAny(env, `/\`) {
		return fmt.Errorf("invalid GOPHER_ENVIRONMENT %q: environment must be a single path segment", env)
	}
	if filepath.Clean(env) != env || filepath.Base(env) != env {
		return fmt.Errorf("invalid GOPHER_ENVIRONMENT %q: environment must be a single path segment", env)
	}
	return nil
}

func setActiveConfigVariables(values *configVariableSet) {
	if values == nil {
		values = newConfigVariableSet()
	}
	activeConfigVariables.Lock()
	activeConfigVariables.values = values
	activeConfigVariables.Unlock()
}

func loadConfigVariables() (*configVariableSet, error) {
	values := newConfigVariableSet()
	if strings.TrimSpace(configPath) == "" {
		return values, nil
	}
	env := currentConfigTemplateEnvironment()
	if err := validateConfigTemplateEnvironment(env); err != nil {
		return nil, err
	}
	for _, filename := range []string{
		filepath.Join("variables", "common.yaml"),
		filepath.Join("variables", env+".yaml"),
	} {
		path := filepath.Join(configPath, "conf", filename)
		if err := mergeConfigVariablesFile(path, values); err != nil {
			return nil, err
		}
	}
	return values, nil
}

func mergeConfigVariablesFile(path string, values *configVariableSet) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading variables file %q: %w", path, err)
	}
	Log(robot.Debug, "Loaded custom variables file %s", path)
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	var loaded configVariablesFile
	if err := dec.Decode(&loaded); err != nil {
		return fmt.Errorf("unmarshalling variables file %q: %w", path, err)
	}
	mergeVariableMap(values.Secrets, loaded.Secrets)
	mergeVariableMap(values.Variables, loaded.Variables)
	return nil
}

func mergeVariableMap(dst map[string]string, src map[string]*string) {
	for k, v := range src {
		if v == nil {
			delete(dst, k)
			continue
		}
		dst[k] = *v
	}
}

// decryptTpl is intentionally retained only to make legacy template use fail
// with an actionable migration message.
func decryptTpl(encval string) (string, error) {
	return "", fmt.Errorf("template function \"decrypt\" was removed in v3; move this encrypted value to custom/conf/variables/common.yaml or custom/conf/variables/<environment>.yaml under Secrets and reference it with {{ secret \"NAME\" }}")
}

// secretTpl resolves a named encrypted secret from custom conf/variables files.
func secretTpl(name string) (string, error) {
	cryptKey.RLock()
	initialized := cryptKey.initialized
	key := cryptKey.key
	cryptKey.RUnlock()
	if !initialized {
		return "", fmt.Errorf("template secret %q requested but encryption is not initialized", name)
	}
	activeConfigVariables.RLock()
	encval, ok := activeConfigVariables.values.Secrets[name]
	activeConfigVariables.RUnlock()
	if !ok {
		return "", fmt.Errorf("template secret %q is not defined in custom conf/variables/common.yaml or custom conf/variables/%s.yaml", name, currentConfigTemplateEnvironment())
	}
	encbytes, err := base64.StdEncoding.DecodeString(encval)
	if err != nil {
		return "", fmt.Errorf("base64 decoding template secret %q: %w", name, err)
	}
	secret, decerr := decrypt(encbytes, key)
	if decerr != nil {
		return "", fmt.Errorf("decrypting template secret %q: %w", name, decerr)
	}
	return string(secret), nil
}

// variableTpl resolves a named plaintext variable from custom conf/variables files.
func variableTpl(name string) (string, error) {
	activeConfigVariables.RLock()
	value, ok := activeConfigVariables.values.Variables[name]
	activeConfigVariables.RUnlock()
	if !ok {
		return "", fmt.Errorf("template variable %q is not defined in custom conf/variables/common.yaml or custom conf/variables/%s.yaml", name, currentConfigTemplateEnvironment())
	}
	return value, nil
}

func isTestBuildTpl() bool {
	return isTestBuild
}

/*
Used in robot.yaml to determine start-up settings for connector, brain, and
logging. Returns one of:
* demo - no configuration or env vars, starts the default robot
* test-dev - using config dir in gopherbot/test for creating integration tests
* bootstrap - env vars (e.g. GOPHER_CUSTOM_REPOSITORY) set, but no config yet
* cli - gopherbot CLI command, not starting a real robot
* production - env vars set and config repo cloned, the most "normal" start-up
*/
func detectStartupMode() (mode string) {
	if cliOp {
		return "cli"
	}
	if _, err := os.Stat(filepath.Join("conf", robotConfigFileName)); err == nil {
		cwd, err := os.Getwd()
		if err != nil {
			panic("Unable to get current directory")
		}
		if !strings.HasSuffix(cwd, "/custom") {
			return "test-dev"
		}
	}
	_, robotConfigured := lookupEnv("GOPHER_CUSTOM_REPOSITORY")
	if !robotConfigured {
		return "demo"
	}
	robotYamlFile := filepath.Join(configPath, "conf", robotConfigFileName)
	if _, err := os.Stat(robotYamlFile); err != nil {
		return "bootstrap"
	}
	return "production"
}

type loadTpl struct {
	dir      string
	isCustom bool
}

func (t loadTpl) Include(tpl string) string {
	base := installPath
	if t.isCustom {
		base = configPath
	}
	path := filepath.Join(base, t.dir, tpl)
	Log(robot.Debug, "Loading Include'd config: %s", path)
	incfile, err := os.Open(path)
	if err != nil {
		Log(robot.Error, "Opening include '%s'(%s): %v", tpl, path, err)
		return ""
	}
	var incbuff bytes.Buffer
	incscanner := bufio.NewScanner(incfile)
	for incscanner.Scan() {
		line := incscanner.Text()
		if !strings.HasPrefix(line, "---") {
			incbuff.WriteString(line)
			incbuff.WriteString("\n")
		}
	}
	err = incscanner.Err()
	if err != nil {
		Log(robot.Error, "Reading include '%s'(%s): %v", tpl, path, err)
		return ""
	}
	inc := incbuff.Bytes()
	expanded, err := expand(t.dir, t.isCustom, inc)
	if err != nil {
		Log(robot.Error, "Expanding included '%s': %v", tpl, err)
		return ""
	}
	return string(expanded)
}

// expand expands a text template
func expand(dir string, custom bool, in []byte) (out []byte, err error) {
	lt := loadTpl{
		dir:      dir,
		isCustom: custom,
	}
	tplFuncs := template.FuncMap{
		"decrypt":        decryptTpl,
		"default":        defval,
		"env":            env,
		"secret":         secretTpl,
		"variable":       variableTpl,
		"GetStartupMode": detectStartupMode,
		"SetEnv":         setEnvVar,
		"IsTestBuild":    isTestBuildTpl,
	}
	var outBuff bytes.Buffer
	tpl, err := template.New("").Funcs(tplFuncs).Parse(string(in))
	if err != nil {
		return nil, err
	}
	if err := tpl.Execute(&outBuff, lt); err != nil {
		return nil, err
	}
	return outBuff.Bytes(), nil
}

// getConfigFile loads a config file first from installPath, then from configPath
// if set. Required indicates whether to return an error if neither file is found.
func getConfigFile(filename string, required bool, jsonMap map[string]json.RawMessage, prev ...map[string]interface{}) error {
	var (
		cf           []byte
		err, realerr error
	)

	loaded := false
	var path string

	var cfg map[string]interface{}
	installed := make(map[string]interface{})
	configured := make(map[string]interface{})
	if len(prev) > 0 && prev[0] != nil {
		cfg = prev[0]
	} else {
		cfg = make(map[string]interface{})
	}
	path = filepath.Join(installPath, "conf", filename)
	// compatibility with old config file name
	if filename == "gopherbot.yaml" {
		Log(robot.Warn, "Merging legacy custom gopherbot.yaml with installed robot.yaml")
		path = filepath.Join(installPath, "conf", "robot.yaml")
	}
	dir := filepath.Dir(filepath.Join("conf", filename))
	cf, err = getConfigFileContent(path, dir, false)
	if err == nil {
		if err = validate_yaml(path, cf); err != nil {
			Log(robot.Error, "Validating installed/default configuration: %v", err)
			return err
		}
		if err = yaml.Unmarshal(cf, &installed); err != nil {
			err = fmt.Errorf("unmarshalling installed \"%s\": %v", filename, err)
			Log(robot.Error, err.Error())
			return err
		}
		if len(installed) == 0 {
			Log(robot.Error, "Empty config hash loading %s", path)
		} else {
			Log(robot.Debug, "Loaded installed conf/%s", filename)
			cfg = mergemap(installed, cfg)
			loaded = true
		}
	} else {
		realerr = err
	}
	if len(configPath) > 0 {
		path = filepath.Join(configPath, "conf", filename)
		cf, err = getConfigFileContent(path, dir, true)
		if err == nil {
			if err = validate_yaml(path, cf); err != nil {
				Log(robot.Error, "Validating configured/custom configuration: %v", err)
				return err
			}
			if err = yaml.Unmarshal(cf, &configured); err != nil {
				err = fmt.Errorf("unmarshalling configured \"%s\": %v", filename, err)
				Log(robot.Error, err.Error())
				return err // If a badly-formatted config is loaded, we always return an error
			}
			if len(configured) == 0 {
				Log(robot.Error, "Empty config hash loading %s", path)
			} else {
				Log(robot.Debug, "Loaded configured conf/%s", filename)
				cfg = mergemap(configured, cfg)
				loaded = true
			}
		} else {
			realerr = err
		}
	}
	jsonData, _ := json.Marshal(cfg)
	json.Unmarshal(jsonData, &jsonMap)
	if required && !loaded {
		return realerr
	}
	return nil
}

// getConfigFileContent is a helper function to read and expand config files
func getConfigFileContent(path, dir string, isCustom bool) ([]byte, error) {
	Log(robot.Debug, "Loading config file: %s", path)
	cf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cf, err = expand(dir, isCustom, cf)
	if err != nil {
		Log(robot.Error, "Expanding '%s': %v", path, err)
		return nil, err
	}
	return cf, nil
}
