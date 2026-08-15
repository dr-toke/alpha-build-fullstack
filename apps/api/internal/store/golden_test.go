package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestAppendFixture(t *testing.T) {
	dir := t.TempDir()
	id := uuid.New()

	t.Run("creates a new fixture file", func(t *testing.T) {
		err := AppendFixture(dir, id, "cbdstore",
			GoldenRaw{Title: "Test Product", Description: "a description"},
			map[string]any{"form": "capsule"}, "first override on this cluster")
		if err != nil {
			t.Fatal(err)
		}

		path := filepath.Join(dir, id.String()+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var fx GoldenFixture
		if err := json.Unmarshal(data, &fx); err != nil {
			t.Fatal(err)
		}
		if fx.Expect["form"] != "capsule" {
			t.Errorf("Expect[form] = %v, want capsule", fx.Expect["form"])
		}
	})

	t.Run("a second call merges into the same file rather than overwriting", func(t *testing.T) {
		err := AppendFixture(dir, id, "cbdstore",
			GoldenRaw{Title: "Test Product", Description: "a description"},
			map[string]any{"route": "oral"}, "second override, different facet")
		if err != nil {
			t.Fatal(err)
		}

		path := filepath.Join(dir, id.String()+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var fx GoldenFixture
		if err := json.Unmarshal(data, &fx); err != nil {
			t.Fatal(err)
		}
		if fx.Expect["form"] != "capsule" {
			t.Errorf("first call's expectation was lost on merge: Expect[form] = %v", fx.Expect["form"])
		}
		if fx.Expect["route"] != "oral" {
			t.Errorf("Expect[route] = %v, want oral", fx.Expect["route"])
		}
		if fx.RegressionNote != "second override, different facet" {
			t.Errorf("RegressionNote = %q, want the latest note", fx.RegressionNote)
		}
	})

	t.Run("a later value for the SAME key overwrites, doesn't error", func(t *testing.T) {
		err := AppendFixture(dir, id, "cbdstore",
			GoldenRaw{Title: "Test Product", Description: "a description"},
			map[string]any{"form": "edible"}, "corrected again")
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, id.String()+".json")
		data, _ := os.ReadFile(path)
		var fx GoldenFixture
		json.Unmarshal(data, &fx)
		if fx.Expect["form"] != "edible" {
			t.Errorf("Expect[form] = %v, want edible (the latest correction should win)", fx.Expect["form"])
		}
	})

	t.Run("invalid directory returns an error, not a panic", func(t *testing.T) {
		err := AppendFixture("/nonexistent/nowhere", uuid.New(), "x", GoldenRaw{}, map[string]any{}, "")
		if err == nil {
			t.Error("expected an error writing to a nonexistent directory, got nil")
		}
	})
}
