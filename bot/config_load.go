package bot

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/lnxjedi/gopherbot/robot"
)

// merge map merges maps and concatenates slices; values in m(erge) override values
// in t(arget).
func mergemap(m, t map[string]interface{}) map[string]interface{} {
	for k, v := range m {
		if tv, ok := t[k]; ok {
			if reflect.TypeOf(v) == reflect.TypeOf(tv) {
				switch v.(type) {
				case map[string]interface{}:
					mv := v.(map[string]interface{})
					mtv := tv.(map[string]interface{})
					t[k] = mergemap(mv, mtv)
				case []interface{}:
					sv := v.([]interface{})
					t[k] = sv
				default:
					t[k] = v
				}
			} else {
				// mis-matched types, use new value
				t[k] = v
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
	val := os.Getenv(envvar)
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

// decryptTpl takes an base64 encoded string, decodes and decrypts, and returns
// the value.
func decryptTpl(encval string) string {
	cryptKey.RLock()
	initialized := cryptKey.initialized
	key := cryptKey.key
	cryptKey.RUnlock()
	if !initialized {
		Log(robot.Warn, "Template called decrypt(Tpl) function but encryption not initialized")
		return ""
	}
	encbytes, err := base64.StdEncoding.DecodeString(encval)
	if err != nil {
		Log(robot.Error, "Unable to base64 decode in template decrypt(Tpl): %v", err)
		return ""
	}
	secret, decerr := decrypt(encbytes, key)
	if decerr != nil {
		Log(robot.Error, "Unable to decrypt secret in template decrypt(Tpl): %v", decerr)
		return ""
	}
	return string(secret)
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
	inc, err := ioutil.ReadFile(path)
	if err != nil {
		Log(robot.Error, "Reading include '%s'(%s): %v", tpl, path, err)
		return ""
	}
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
		"decrypt": decryptTpl,
		"default": defval,
		"env":     env,
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
	dir := filepath.Dir(filepath.Join("conf", filename))
	cf, err = ioutil.ReadFile(path)
	if err == nil {
		if cf, err = expand(dir, false, cf); err != nil {
			Log(robot.Error, "Expanding '%s': %v", path, err)
		}
		if err = yaml.Unmarshal(cf, &installed); err != nil {
			err = fmt.Errorf("Unmarshalling installed \"%s\": %v", filename, err)
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
		cf, err = ioutil.ReadFile(path)
		if err == nil {
			if cf, err = expand(dir, true, cf); err != nil {
				Log(robot.Error, "Expanding '%s': %v", path, err)
			}
			if err = yaml.Unmarshal(cf, &configured); err != nil {
				err = fmt.Errorf("Unmarshalling configured \"%s\": %v", filename, err)
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
