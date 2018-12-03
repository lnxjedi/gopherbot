package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"text/template"

	"github.com/ghodss/yaml"
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
					stv := tv.([]interface{})
					stv = append(stv, sv...)
					t[k] = stv
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

// NOTE: even after calling os.Unsetenv() or os.Setenv(), the original
// values can be found in /proc/<pid>/environ, removing any benefit.
// For now, leaving the old code, should probably eventually be removed.

/*
var envrmcache = struct {
	env map[string]string
	*sync.Mutex
}{
	env:   make(map[string]string),
	Mutex: new(sync.Mutex),
}

// envrm is for the config file template FuncMap. It returns
// the given environment var if found, then unsets it; best used
// for security-sensitive values that shouldn't remain in the
// environment. If required is set, an Error-level log event is
// generated for empty vars.
func envrm(envvar string) (val string) {
	envrmcache.Lock()
	if cached, ok := envrmcache.env[envvar]; ok {
		envrmcache.Unlock()
		val = cached
	} else {
		val = os.Getenv(envvar)
		envrmcache.env[envvar] = val
		envrmcache.Unlock()
		err := os.Unsetenv(envvar)
		if err != nil {
			Log(Debug, fmt.Sprintf("Error unsetting environment variable '%s': %v", envvar, err))
		}
	}
	if len(val) == 0 {
		Log(Debug, fmt.Sprintf("Empty environemnt variable returned for '%s' in template expansion", envvar))
	}
	return
}
*/

// env is for the config file template FuncMap. It returns
// the given environment var if found. If required is set,
// an Error-level log event is generated for empty vars.
func env(envvar string) string {
	val := os.Getenv(envvar)
	if len(val) == 0 {
		Log(Debug, fmt.Sprintf("Empty environemnt variable returned for '%s' in template expansion", envvar))
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

// expand expands a text template
func expand(in []byte) (out []byte, err error) {
	tplFuncs := template.FuncMap{
		"default": defval,
		"env":     env,
	}
	var outBuff bytes.Buffer
	tpl, err := template.New("").Funcs(tplFuncs).Parse(string(in))
	if err != nil {
		return nil, err
	}
	if err := tpl.Execute(&outBuff, nil); err != nil {
		return nil, err
	}
	return outBuff.Bytes(), nil
}

// getConfigFile loads a config file first from installPath, then from configPath
// if set. Required indicates whether to return an error if neither file is found.
func (c *botContext) getConfigFile(filename, callerID string, required bool, jsonMap map[string]json.RawMessage, prev ...map[string]interface{}) error {
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
	path = installPath + "/conf/" + filename
	cf, err = ioutil.ReadFile(path)
	if err == nil {
		if cf, err = expand(cf); err != nil {
			err = fmt.Errorf("Expanding '%s': %v", path, err)
			Log(Error, err)
		}
		if err = yaml.Unmarshal(cf, &installed); err != nil {
			err = fmt.Errorf("Unmarshalling installed \"%s\": %v", filename, err)
			Log(Error, err)
			return err
		}
		if len(installed) == 0 {
			Log(Error, fmt.Sprintf("Empty config hash loading %s", path))
		} else {
			Log(Debug, fmt.Sprintf("Loaded installed conf/%s", filename))
			cfg = mergemap(installed, cfg)
			loaded = true
		}
	} else {
		realerr = err
	}
	if len(configPath) > 0 {
		path = configPath + "/conf/" + filename
		cf, err = ioutil.ReadFile(path)
		if err == nil {
			if err = yaml.Unmarshal(cf, &configured); err != nil {
				err = fmt.Errorf("Unmarshalling configured \"%s\": %v", filename, err)
				Log(Error, err)
				return err // If a badly-formatted config is loaded, we always return an error
			}
			if len(configured) == 0 {
				Log(Error, fmt.Sprintf("Empty config hash loading %s", path))
			} else {
				Log(Debug, fmt.Sprintf("Loaded configured conf/%s", filename))
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
