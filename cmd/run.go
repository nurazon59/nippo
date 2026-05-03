package cmd

import (
	"github.com/adrg/xdg"
	"github.com/alecthomas/kong"
	template "github.com/nurazon59/go-template"
)

const appVersion = "v0.1.0"

var CLI struct {
	Config  string           `help:"Path to config file." env:"GO_TEMPLATE_CONFIG"`
	Version kong.VersionFlag `name:"version" help:"Print version information and quit."`
}

func Run() error {
	kong.Parse(&CLI, kong.Name("go-template"), kong.Vars{
		"version": appVersion,
	})

	configPath := CLI.Config
	if configPath == "" {
		var err error
		configPath, err = xdg.ConfigFile("go-template/config.yaml")
		if err != nil {
			return err
		}
	}

	_, err := template.Load(configPath)
	if err != nil {
		return err
	}

	return nil
}
