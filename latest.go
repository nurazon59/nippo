package main

import (
	"errors"
	"fmt"
	"io/fs"
)

type latestCmd struct{}

func (c *latestCmd) Run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	storage, err := NewStorage(cfg)
	if err != nil {
		return err
	}
	defer storage.Close()

	date, err := storage.LoadLatestReport()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return errors.New("no saved reports found")
		}
		return err
	}

	content, err := storage.LoadReport(date)
	if err != nil {
		return err
	}

	fmt.Print(content)
	return nil
}
