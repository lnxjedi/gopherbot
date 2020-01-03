// +build modular

package bot

import (
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
func loadModules(protocol string, modules []LoadableModule) {
	for _, m := range modules {
		loadModule(m.Name, m.Path)
	}
	_, pmod := getProtocol(protocol)
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
		Log(robot.Info, "Loaded module '%s': %s", name, path)
		// look for manifest and register everything
		if gp, err := k.Lookup("GetManifest"); err == nil {
			gf := gp.(func() robot.Manifest)
			pm := gf()
			for _, tspec := range pm.Tasks {
				Log(robot.Info, "Registering task '%s' from loadable module '%s'", tspec.Name, path)
				RegisterTask(tspec.Name, tspec.RequiresPrivilege, tspec.Handler)
			}
			for _, pspec := range pm.Plugins {
				Log(robot.Info, "Registering plugin '%s' from loadable module '%s'", pspec.Name, path)
				RegisterPlugin(pspec.Name, pspec.Handler)
			}
			for _, jspec := range pm.Jobs {
				Log(robot.Info, "Registering job '%s' from loadable module '%s'", jspec.Name, path)
				RegisterJob(jspec.Name, jspec.Handler)
			}
			if len(pm.Connector.Name) > 0 && pm.Connector.Connector != nil {
				Log(robot.Info, "Registering connector '%s' from loadable module '%s'", name, path)
				RegisterConnector(pm.Connector.Name, pm.Connector.Connector)
			}
			if len(pm.Brain.Name) > 0 && pm.Brain.Brain != nil {
				Log(robot.Info, "Registering brain '%s' from loadable module '%s'", name, path)
				RegisterSimpleBrain(pm.Brain.Name, pm.Brain.Brain)
			}
			if len(pm.History.Name) > 0 && pm.History.Provider != nil {
				Log(robot.Info, "Registering history '%s' from loadable module '%s'", name, path)
				RegisterHistoryProvider(pm.History.Name, pm.History.Provider)
			}
		} else {
			Log(robot.Debug, "Symbol 'GetManifest' not found in loadable module '%s'", path)
		}
	} else {
		Log(robot.Error, "Loading module '%s': %v", lp, err)
	}
}
