package main

import (
	"github.com/adrg/xdg"
	"github.com/alecthomas/kong"
)

const appVersion = "v0.1.0"

var CLI struct {
	Config  string           `help:"Path to config file." env:"NIPPO_CONFIG"`
	Version kong.VersionFlag `name:"version" help:"Print version information and quit."`

	Generate   generateCmd   `cmd:"" default:"1" help:"Generate and display a daily report."`
	List       listCmd       `cmd:"" help:"List saved reports."`
	Show       showCmd       `cmd:"" help:"Show report for a specific date."`
	Latest     latestCmd     `cmd:"" help:"Show the most recently dated report."`
	Edit       editCmd       `cmd:"" help:"Edit report for a specific date."`
	Completion completionCmd `cmd:"" help:"Generate shell completion script."`
}

func Run() error {
	ctx := kong.Parse(&CLI,
		kong.Name("nippo"),
		kong.Description("Daily report generator"),
		kong.Vars{"version": appVersion},
	)
	return ctx.Run()
}

func loadConfig() (*Config, error) {
	configPath := CLI.Config
	if configPath == "" {
		var err error
		configPath, err = xdg.ConfigFile("nippo/config.yaml")
		if err != nil {
			return nil, err
		}
	}
	return Load(configPath)
}
