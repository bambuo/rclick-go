package services

import (
	"fmt"
	"strings"

	"github.com/bambuo/rclick-go/internal/config"
	"github.com/bambuo/rclick-go/internal/ipc"
)

type AppLauncher struct {
	Store  *config.Store
	Helper *ipc.Helper
}

func (a *AppLauncher) Open(appID string, targets []string) error {
	cfg := a.Store.Current()
	var app *config.OpenWithApp
	for i := range cfg.Apps {
		if cfg.Apps[i].ID == appID {
			app = &cfg.Apps[i]
			break
		}
	}
	if app == nil {
		return fmt.Errorf("app %q not found", appID)
	}
	if len(targets) == 0 {
		return nil
	}

	var arguments []string
	if app.InheritFromGlobalArguments {
		arguments = append(arguments, splitArguments(cfg.Defaults.GlobalApplicationArguments)...)
	}
	arguments = append(arguments, app.Arguments...)

	environment := map[string]string{}
	if app.InheritFromGlobalEnvironment {
		for key, value := range parsePairs(cfg.Defaults.GlobalApplicationEnvironment) {
			environment[key] = value
		}
	}
	for key, value := range app.Environment {
		environment[key] = value
	}

	if a.Helper == nil {
		return ErrHelperUnavailable
	}
	return a.Helper.OpenWith(app.URL, targets, arguments, environment)
}

func splitArguments(text string) []string {
	return strings.Fields(text)
}

func parsePairs(text string) map[string]string {
	out := map[string]string{}
	for _, token := range strings.Fields(text) {
		index := strings.IndexByte(token, '=')
		if index <= 0 || index == len(token)-1 {
			continue
		}
		out[token[:index]] = token[index+1:]
	}
	return out
}
