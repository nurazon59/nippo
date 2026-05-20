package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunHooks(t *testing.T) {
	date := time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC)

	t.Run("captures stdout into commented preset", func(t *testing.T) {
		hooks := []HookConfig{{
			Name:    "echo",
			Command: "echo from-hook",
			Keys:    []string{"done"},
		}}
		out := RunHooks(context.Background(), hooks, date)
		assert.Equal(t, "<!--\nfrom-hook\n-->", out["done"])
	})

	t.Run("single hook fans out to multiple keys", func(t *testing.T) {
		hooks := []HookConfig{{
			Name:    "echo",
			Command: "echo same",
			Keys:    []string{"done", "todo"},
		}}
		out := RunHooks(context.Background(), hooks, date)
		assert.Equal(t, "<!--\nsame\n-->", out["done"])
		assert.Equal(t, "<!--\nsame\n-->", out["todo"])
	})

	t.Run("timeout drops entry but other hooks survive", func(t *testing.T) {
		hooks := []HookConfig{
			{Name: "slow", Command: "sleep 5", Keys: []string{"done"}, Timeout: "100ms"},
			{Name: "fast", Command: "echo fast", Keys: []string{"todo"}},
		}
		out := RunHooks(context.Background(), hooks, date)
		_, ok := out["done"]
		assert.False(t, ok)
		assert.Equal(t, "<!--\nfast\n-->", out["todo"])
	})

	t.Run("non-zero exit drops entry but other hook still runs", func(t *testing.T) {
		hooks := []HookConfig{
			{Name: "fail", Command: "exit 1", Keys: []string{"done"}},
			{Name: "ok", Command: "echo ok", Keys: []string{"todo"}},
		}
		out := RunHooks(context.Background(), hooks, date)
		_, ok := out["done"]
		assert.False(t, ok)
		assert.Equal(t, "<!--\nok\n-->", out["todo"])
	})

	t.Run("hooks run in parallel", func(t *testing.T) {
		hooks := []HookConfig{
			{Name: "a", Command: "sleep 1 && echo a", Keys: []string{"done"}},
			{Name: "b", Command: "sleep 1 && echo b", Keys: []string{"todo"}},
		}
		start := time.Now()
		out := RunHooks(context.Background(), hooks, date)
		elapsed := time.Since(start)
		assert.Less(t, elapsed, 1800*time.Millisecond)
		assert.Equal(t, "<!--\na\n-->", out["done"])
		assert.Equal(t, "<!--\nb\n-->", out["todo"])
	})

	t.Run("NIPPO_DATE env var is exposed", func(t *testing.T) {
		hooks := []HookConfig{{
			Name:    "env",
			Command: `printf "%s" "$NIPPO_DATE"`,
			Keys:    []string{"done"},
		}}
		out := RunHooks(context.Background(), hooks, date)
		assert.Equal(t, "<!--\n2024-06-16\n-->", out["done"])
	})

	t.Run("multiple hooks targeting same key are concatenated in declaration order", func(t *testing.T) {
		hooks := []HookConfig{
			{Name: "first", Command: "echo one", Keys: []string{"done"}},
			{Name: "second", Command: "echo two", Keys: []string{"done"}},
		}
		out := RunHooks(context.Background(), hooks, date)
		assert.Equal(t, "<!--\none\n-->\n\n<!--\ntwo\n-->", out["done"])
	})
}

func TestMergePresets(t *testing.T) {
	t.Run("both present concatenated", func(t *testing.T) {
		merged := mergePresets(
			map[string]string{"done": "<!--\nprev\n-->"},
			map[string]string{"done": "<!--\nhook\n-->"},
		)
		assert.Equal(t, "<!--\nprev\n-->\n\n<!--\nhook\n-->", merged["done"])
	})

	t.Run("only hook output", func(t *testing.T) {
		merged := mergePresets(
			map[string]string{},
			map[string]string{"done": "<!--\nhook\n-->"},
		)
		assert.Equal(t, "<!--\nhook\n-->", merged["done"])
	})

	t.Run("only preset", func(t *testing.T) {
		merged := mergePresets(
			map[string]string{"done": "<!--\nprev\n-->"},
			map[string]string{},
		)
		assert.Equal(t, "<!--\nprev\n-->", merged["done"])
	})

	t.Run("nil maps are handled", func(t *testing.T) {
		merged := mergePresets(nil, nil)
		assert.Empty(t, merged)

		merged = mergePresets(nil, map[string]string{"done": "<!--\nhook\n-->"})
		assert.Equal(t, "<!--\nhook\n-->", merged["done"])
	})

	t.Run("empty preset value is replaced by hook", func(t *testing.T) {
		merged := mergePresets(
			map[string]string{"done": ""},
			map[string]string{"done": "<!--\nhook\n-->"},
		)
		assert.Equal(t, "<!--\nhook\n-->", merged["done"])
	})
}
