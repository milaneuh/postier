package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"postier/internal/config"
)

// isolate points POSTIER_CONFIG at a file inside a fresh temp directory and
// returns its path. The parent directories deliberately do not exist, so Save
// has to create them.
func isolate(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "nested", "deeper", "config.json")
	t.Setenv("POSTIER_CONFIG", p)
	return p
}

func TestPathPrecedence(t *testing.T) {
	t.Run("POSTIER_CONFIG wins", func(t *testing.T) {
		t.Setenv("POSTIER_CONFIG", "/tmp/explicit.json")
		t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")

		got, err := config.Path()
		if err != nil {
			t.Fatal(err)
		}
		if got != "/tmp/explicit.json" {
			t.Errorf("Path() = %q, want /tmp/explicit.json", got)
		}
	})

	t.Run("XDG_CONFIG_HOME is next", func(t *testing.T) {
		t.Setenv("POSTIER_CONFIG", "")
		t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")

		got, err := config.Path()
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join("/tmp/xdg", "postier", "config.json")
		if got != want {
			t.Errorf("Path() = %q, want %q", got, want)
		}
	})

	t.Run("falls back to home", func(t *testing.T) {
		t.Setenv("POSTIER_CONFIG", "")
		t.Setenv("XDG_CONFIG_HOME", "")

		got, err := config.Path()
		if err != nil {
			t.Fatal(err)
		}
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("no home directory in this environment")
		}
		want := filepath.Join(home, ".config", "postier", "config.json")
		if got != want {
			t.Errorf("Path() = %q, want %q", got, want)
		}
		// A literal "~" means the home directory was never expanded.
		if strings.Contains(got, "~") {
			t.Errorf("Path() = %q, contains an unexpanded ~", got)
		}
	})

	t.Run("does not touch the filesystem", func(t *testing.T) {
		p := isolate(t)

		if _, err := config.Path(); err != nil {
			t.Fatalf("Path() errored on a missing file: %v", err)
		}
		if _, err := os.Stat(filepath.Dir(p)); err == nil {
			t.Error("Path() created the config directory; it should be pure")
		}
	})
}

func TestLoadMissingFileIsNotAnError(t *testing.T) {
	isolate(t)

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load() on a fresh install: %v", err)
	}
	if got == nil {
		t.Fatal("Load() returned a nil Config with a nil error")
	}
	if got.URL != "" || got.Token != "" {
		t.Errorf("Load() = %+v, want zero URL and Token", got)
	}
	if got.Version != config.Version {
		t.Errorf("Version = %d, want %d", got.Version, config.Version)
	}
}

func TestLoadCorruptFileIsAnError(t *testing.T) {
	p := isolate(t)

	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() on a corrupt file: want an error, got nil")
	}
	// The message has to name the file, or the user cannot find what to delete.
	if !strings.Contains(err.Error(), p) {
		t.Errorf("error %q does not name the config path %q", err, p)
	}
}

// A read failure that is NOT "file missing" must surface. Otherwise a
// permission problem or a bad path silently looks like a fresh install, and the
// user gets "not connected" with no idea why.
func TestLoadUnreadableFileIsAnError(t *testing.T) {
	p := isolate(t)

	// A directory where the file should be: os.ReadFile fails with EISDIR,
	// which is a read error but not fs.ErrNotExist.
	if err := os.MkdirAll(p, 0o700); err != nil {
		t.Fatal(err)
	}

	got, err := config.Load()
	if err == nil {
		t.Fatalf("Load() on an unreadable path: want an error, got nil (config %+v)", got)
	}
	if !strings.Contains(err.Error(), p) {
		t.Errorf("error %q does not name the config path %q", err, p)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	isolate(t)

	want := &config.Config{
		URL:     "https://clip.example.dev",
		Token:   "k7f3m9x2qp4w8n6v",
		Version: 0, // Save is responsible for stamping this
	}
	if err := want.Save(); err != nil {
		t.Fatalf("Save(): %v", err)
	}

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if got.URL != want.URL {
		t.Errorf("URL = %q, want %q", got.URL, want.URL)
	}
	if got.Token != want.Token {
		t.Errorf("Token = %q, want %q", got.Token, want.Token)
	}
	if got.Version != config.Version {
		t.Errorf("Version = %d, want %d — Save must stamp it", got.Version, config.Version)
	}
}

func TestSaveCreatesParentDirectories(t *testing.T) {
	p := isolate(t)

	if err := (&config.Config{URL: "https://example.dev"}).Save(); err != nil {
		t.Fatalf("Save() into a missing directory tree: %v", err)
	}

	fi, err := os.Stat(p)
	if err != nil {
		t.Fatalf("stat after Save(): %v", err)
	}
	// Guards the MkdirAll(configPath) bug, which made config.json a directory.
	if fi.IsDir() {
		t.Fatal("config path is a directory; MkdirAll must take filepath.Dir")
	}
}

func TestSavePermissions(t *testing.T) {
	p := isolate(t)

	if err := (&config.Config{Token: "secret"}).Save(); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %#o, want 0600 — it holds a bearer token", perm)
	}

	di, err := os.Stat(filepath.Dir(p))
	if err != nil {
		t.Fatal(err)
	}
	if perm := di.Mode().Perm(); perm != 0o700 {
		t.Errorf("directory mode = %#o, want 0700", perm)
	}
}

func TestSaveOverwrites(t *testing.T) {
	isolate(t)

	if err := (&config.Config{URL: "https://first.dev", Token: "a"}).Save(); err != nil {
		t.Fatal(err)
	}
	if err := (&config.Config{URL: "https://second.dev", Token: "b"}).Save(); err != nil {
		t.Fatal(err)
	}

	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "https://second.dev" || got.Token != "b" {
		t.Errorf("after overwrite = %+v, want the second config", got)
	}
}

func TestResolve(t *testing.T) {
	const builtIn = "http://localhost:8080"

	tests := []struct {
		name      string
		file      *config.Config
		envURL    string
		envToken  string
		flagURL   string
		flagToken string
		wantURL   string
		wantToken string
	}{
		{
			name:    "built-in default when nothing is set",
			file:    &config.Config{},
			wantURL: builtIn,
		},
		{
			name:      "file beats the built-in default",
			file:      &config.Config{URL: "https://file.dev", Token: "file-tok"},
			wantURL:   "https://file.dev",
			wantToken: "file-tok",
		},
		{
			name:      "env beats the file",
			file:      &config.Config{URL: "https://file.dev", Token: "file-tok"},
			envURL:    "https://env.dev",
			envToken:  "env-tok",
			wantURL:   "https://env.dev",
			wantToken: "env-tok",
		},
		{
			name:      "flag beats the env",
			file:      &config.Config{URL: "https://file.dev", Token: "file-tok"},
			envURL:    "https://env.dev",
			envToken:  "env-tok",
			flagURL:   "https://flag.dev",
			flagToken: "flag-tok",
			wantURL:   "https://flag.dev",
			wantToken: "flag-tok",
		},
		{
			name:      "layers are independent",
			file:      &config.Config{URL: "https://file.dev", Token: "file-tok"},
			flagURL:   "https://flag.dev",
			wantURL:   "https://flag.dev",
			wantToken: "file-tok",
		},
		{
			name:      "empty values do not override",
			file:      &config.Config{URL: "https://file.dev", Token: "file-tok"},
			envURL:    "",
			flagURL:   "",
			wantURL:   "https://file.dev",
			wantToken: "file-tok",
		},
		{
			name:      "env can supply a token the file lacks",
			file:      &config.Config{URL: "https://file.dev"},
			envToken:  "env-tok",
			wantURL:   "https://file.dev",
			wantToken: "env-tok",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("POSTIER_URL", tc.envURL)
			t.Setenv("POSTIER_TOKEN", tc.envToken)

			got := config.Resolve(tc.file, tc.flagURL, tc.flagToken, builtIn)

			if got.URL != tc.wantURL {
				t.Errorf("URL = %q, want %q", got.URL, tc.wantURL)
			}
			if got.Token != tc.wantToken {
				t.Errorf("Token = %q, want %q", got.Token, tc.wantToken)
			}
		})
	}
}

func TestResolveHandlesNilConfig(t *testing.T) {
	t.Setenv("POSTIER_URL", "")
	t.Setenv("POSTIER_TOKEN", "")

	// A nil file config must behave like an empty one, not panic.
	got := config.Resolve(nil, "", "", "http://localhost:8080")

	if got == nil {
		t.Fatal("Resolve(nil, ...) returned nil")
	}
	if got.URL != "http://localhost:8080" {
		t.Errorf("URL = %q, want the built-in default", got.URL)
	}
	if got.Token != "" {
		t.Errorf("Token = %q, want empty", got.Token)
	}
}

func TestResolveDoesNotMutateItsInput(t *testing.T) {
	t.Setenv("POSTIER_URL", "https://env.dev")

	file := &config.Config{URL: "https://file.dev", Token: "file-tok"}
	config.Resolve(file, "https://flag.dev", "flag-tok", "http://localhost:8080")

	if file.URL != "https://file.dev" || file.Token != "file-tok" {
		t.Errorf("Resolve mutated the caller's Config: %+v", file)
	}
}
