package bot

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type ConfigDuration struct {
	duration time.Duration
	set      bool
}

func (d *ConfigDuration) UnmarshalJSON(data []byte) error {
	return d.decodeDuration(func(target interface{}) error {
		return json.Unmarshal(data, target)
	})
}

func (d *ConfigDuration) UnmarshalYAML(value *yaml.Node) error {
	return d.decodeDuration(value.Decode)
}

func (d *ConfigDuration) decodeDuration(decode func(target interface{}) error) error {
	d.set = true
	var raw string
	if err := decode(&raw); err == nil {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			d.duration = 0
			return nil
		}
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", raw, err)
		}
		if parsed < 0 {
			return fmt.Errorf("duration must not be negative")
		}
		d.duration = parsed
		return nil
	}
	var nanos int64
	if err := decode(&nanos); err == nil {
		if nanos < 0 {
			return fmt.Errorf("duration must not be negative")
		}
		d.duration = time.Duration(nanos)
		return nil
	}
	var nullable interface{}
	if err := decode(&nullable); err == nil && nullable == nil {
		d.duration = 0
		return nil
	}
	return fmt.Errorf("duration must be a quoted duration string or integer nanoseconds")
}

func (d ConfigDuration) Duration() time.Duration {
	return d.duration
}

func (d ConfigDuration) IsSet() bool {
	return d.set
}

type TimeOutThresholds struct {
	Warn ConfigDuration `yaml:"Warn"`
	Kill ConfigDuration `yaml:"Kill"`
}

type TimeOutsConfig struct {
	Plugin TimeOutThresholds `yaml:"Plugin"`
	Job    TimeOutThresholds `yaml:"Job"`
}

type runtimeTimeOutThresholds struct {
	Warn time.Duration
	Kill time.Duration
}

type runtimeTimeOutsConfig struct {
	Plugin runtimeTimeOutThresholds
	Job    runtimeTimeOutThresholds
}

func (t TimeOutThresholds) runtime() runtimeTimeOutThresholds {
	return runtimeTimeOutThresholds{
		Warn: t.Warn.Duration(),
		Kill: t.Kill.Duration(),
	}
}

func (t TimeOutsConfig) runtime() runtimeTimeOutsConfig {
	return runtimeTimeOutsConfig{
		Plugin: t.Plugin.runtime(),
		Job:    t.Job.runtime(),
	}
}

func validateTimeOutThresholds(scope string, cfg TimeOutThresholds) error {
	return validateRuntimeTimeOutThresholds(scope, cfg.runtime())
}

func validateRuntimeTimeOutThresholds(scope string, cfg runtimeTimeOutThresholds) error {
	warn := cfg.Warn
	kill := cfg.Kill
	if warn < 0 {
		return fmt.Errorf("%s Warn must not be negative", scope)
	}
	if kill < 0 {
		return fmt.Errorf("%s Kill must not be negative", scope)
	}
	if warn > 0 && kill > 0 && kill <= warn {
		return fmt.Errorf("%s Kill must be greater than Warn when both are set", scope)
	}
	return nil
}

func resolveTimeOutThresholds(defaults runtimeTimeOutThresholds, overrides TimeOutThresholds) runtimeTimeOutThresholds {
	resolved := defaults
	if overrides.Warn.IsSet() {
		resolved.Warn = overrides.Warn.Duration()
	}
	if overrides.Kill.IsSet() {
		resolved.Kill = overrides.Kill.Duration()
	}
	return resolved
}

func (t runtimeTimeOutThresholds) any() bool {
	return t.Warn > 0 || t.Kill > 0
}
