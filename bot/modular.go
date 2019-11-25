// +build modular

package bot

import (
	"log"
	"path/filepath"
	"plugin"

	"github.com/lnxjedi/gopherbot/robot"
)

// Loadable modules can still be compiled in to the binary. When
// this is the case, the module should register as preloaded during
// init, to prevent automatic loading of the module.
var preloaded map[string]struct{} = make(map[string]struct{})

func RegisterPreload(mod string) {
	preloaded[mod] = struct{}{}
}

// Load pluggable modules and call "GetPlugins", "GetConnectors", etc., then
// register them.
func loadModules() {
	for _, m := range botCfg.loadableModules {
		loadModule(m.Name, m.Path)
	}
	_, pmod := getProtocol(botCfg.protocol)
	ppath := filepath.Join("connectors", pmod+".so")
	loadModule(pmod, ppath)
}

// loadModule loads a module and registers it's contents
func loadModule(name, path string) {
	if _, ok := preloaded[path]; ok {
		Log(robot.Debug, "Skipping load of already loaded or compiled in module: %s", path)
		return
	}
	preloaded[path] = struct{}{}
	lp, err := getObjectPath(path)
	if err != nil {
		Log(robot.Warn, "Unable to locate loadable module '%s' from path '%s'", name, path)
		return
	}
	if k, err := plugin.Open(lp); err == nil {
		Log(robot.Info, "Loaded module '%s': %s", name, path						)
		// look for and register plugins
		if gp, err := k.Lookup("GetPlugins"); err == nil {
			gf := gp.(func() []robot.PluginSpec)
			pl := gf()
			for _, pspec := range pl {
				Log(robot.Info, "Registering plugin '%s' from loadable module '%s'", pspec.Name, path)
				RegisterPlugin(pspec.Name, pspec.Handler)
			}
		} else {
			Log(robot.Debug, "Symbol 'GetPlugins' not found in loadable module '%s': %v", path, err)
		}
		// look for and register connector
		if ci, err := k.Lookup("GetInitializer"); err == nil {
			cif := ci.(func() (string, func(robot.Handler, *log.Logger) robot.Connector))
			name, initializer := cif()
			Log(robot.Info, "Registering connector '%s' from loadable module '%s'", name, path)
			RegisterConnector(name, initializer)
		} else {
			Log(robot.Debug, "Symbol 'GetInitializer' not found in loadable module '%s': %v", path, err)
		}
		// look for and register brain
		if bp, err := k.Lookup("GetBrainProvider"); err == nil {
			bpf := bp.(func() (string, func(robot.Handler) robot.SimpleBrain))
			name, provider := bpf()
			Log(robot.Info, "Registering brain provider '%s' from loadable module '%s'", name, path)
			RegisterSimpleBrain(name, provider)
		} else {
			Log(robot.Debug, "Symbol 'GetInitializer' not found in loadable module '%s': %v", path, err)
		}
	} else {
		Log(robot.Error, "Loading module '%s': %v", lp, err)
	}
}
