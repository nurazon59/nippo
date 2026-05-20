package main

import (
	"errors"
	"fmt"
	"io/fs"
	"time"
)

type showCmd struct {
	Date string `arg:"" required:"" help:"Target date (YYYY-MM-DD)."`
}

func (c *showCmd) Run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	date, err := time.Parse("2006-01-02", c.Date)
	if err != nil {
		return err
	}

	storage, err := NewStorage(cfg)
	if err != nil {
		return err
	}
	defer storage.Close()

	content, err := storage.LoadReport(date)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("no report found for %s", c.Date)
		}
		return err
	}

	fmt.Print(content)
	return nil
}
