package bot

import "github.com/lnxjedi/gopherbot/robot"

var brainProviderRegistrationOverrides = map[string]robot.BrainProviderRegistration{}
var historyProviderRegistrationOverrides = map[string]robot.HistoryProviderRegistration{}

func brainProviderRegistration(name string) (robot.BrainProviderRegistration, bool) {
	if registration, ok := brainProviderRegistrationOverrides[name]; ok {
		return registration, true
	}
	return robot.GetBrainProviderRegistration(name)
}

func historyProviderRegistration(name string) (robot.HistoryProviderRegistration, bool) {
	if registration, ok := historyProviderRegistrationOverrides[name]; ok {
		return registration, true
	}
	return robot.GetHistoryProviderRegistration(name)
}
