package main

import (
	"fmt"
	"time"
)

type editCmd struct {
	Date       string                                    `arg:"" help:"Target date (YYYY-MM-DD)."`
	openEditor func(cmd, content string) (string, error) `kong:"-"`
}

func (c *editCmd) Run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	date, err := time.Parse("2006-01-02", c.Date)
	if err != nil {
		return fmt.Errorf("invalid date format: %w", err)
	}

	storage, err := NewStorage(cfg)
	if err != nil {
		return err
	}
	defer storage.Close()

	existing, err := storage.LoadReport(date)
	if err != nil {
		return fmt.Errorf("report not found for %s: %w", c.Date, err)
	}

	openEditor := c.openEditor
	if openEditor == nil {
		openEditor = launchEditor
	}

	edited, err := openEditor("", existing)
	if err != nil {
		return err
	}

	if edited == existing {
		fmt.Println("No changes made.")
		return nil
	}

	if err := storage.SaveReport(edited, date); err != nil {
		return err
	}

	fmt.Println("Report updated successfully.")
	return nil
}
