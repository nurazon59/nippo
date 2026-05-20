package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const defaultHookTimeout = 30 * time.Second

func RunHooks(ctx context.Context, hooks []HookConfig, date time.Time) map[string]string {
	type result struct {
		hook   HookConfig
		output string
	}

	results := make([]result, len(hooks))
	var wg sync.WaitGroup
	for i, h := range hooks {
		wg.Add(1)
		go func(i int, h HookConfig) {
			defer wg.Done()
			output, err := runHook(ctx, h, date)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[hook:%s] %v\n", h.Name, err)
				return
			}
			results[i] = result{hook: h, output: output}
		}(i, h)
	}
	wg.Wait()

	out := make(map[string]string)
	for _, r := range results {
		preset := commentOutPreset(r.output)
		if preset == "" {
			continue
		}
		for _, k := range r.hook.Keys {
			if existing, ok := out[k]; ok && existing != "" {
				out[k] = existing + "\n\n" + preset
			} else {
				out[k] = preset
			}
		}
	}
	return out
}

func runHook(ctx context.Context, h HookConfig, date time.Time) (string, error) {
	timeout := defaultHookTimeout
	if h.Timeout != "" {
		d, err := time.ParseDuration(h.Timeout)
		if err != nil {
			return "", fmt.Errorf("invalid timeout %q: %w", h.Timeout, err)
		}
		timeout = d
	}

	hookCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(hookCtx, "sh", "-c", h.Command)
	cmd.Env = append(os.Environ(), "NIPPO_DATE="+date.Format("2006-01-02"))
	var stderr strings.Builder
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if errors.Is(hookCtx.Err(), context.DeadlineExceeded) {
		return "", fmt.Errorf("timeout after %s", timeout)
	}
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return "", fmt.Errorf("%w: %s", err, msg)
		}
		return "", err
	}
	return string(stdout), nil
}

func mergePresets(presets, hookOut map[string]string) map[string]string {
	merged := make(map[string]string, len(presets)+len(hookOut))
	for k, v := range presets {
		merged[k] = v
	}
	for k, v := range hookOut {
		if existing, ok := merged[k]; ok && existing != "" {
			merged[k] = existing + "\n\n" + v
		} else {
			merged[k] = v
		}
	}
	return merged
}
