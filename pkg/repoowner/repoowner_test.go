package repoowner

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, baseDir, platform, repo, body string) {
	t.Helper()
	dir := filepath.Join(baseDir, "repos", platform, filepath.FromSlash(repo))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestResolve(t *testing.T) {
	base := t.TempDir()
	platform := "bitbucket-server"
	r := &Resolver{BaseDir: base, Platform: platform}

	writeConfig(t, base, platform, "fnx/currencies", `{"serviceName":"currencies","team":"gonix"}`)
	writeConfig(t, base, platform, "gl/no-team", `{"serviceName":"cache"}`)
	writeConfig(t, base, platform, "gl/broken", `{not json`)

	cases := []struct {
		name       string
		repo       string
		wantTeam   string
		wantSvcNam string
	}{
		{"full config", "fnx/currencies", "gonix", "currencies"},
		{"strips options", "fnx/currencies?loglevel=debug", "gonix", "currencies"},
		{"missing team falls back, keeps serviceName", "gl/no-team", Unknown, "cache"},
		{"broken json falls back to slug", "gl/broken", Unknown, "gl/broken"},
		{"missing file falls back to slug", "gl/does-not-exist", Unknown, "gl/does-not-exist"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			team, name := r.Resolve(tc.repo)
			if team != tc.wantTeam {
				t.Errorf("team = %q, want %q", team, tc.wantTeam)
			}
			if name != tc.wantSvcNam {
				t.Errorf("name = %q, want %q", name, tc.wantSvcNam)
			}
		})
	}
}
