package repoowner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

const Unknown = "unknown"

// repoConfig is the subset of a repo's root config.json used for alert routing
type repoConfig struct {
	ServiceName string `json:"serviceName"`
	Team        string `json:"team"`
}

// Resolver looks up owner metadata from renovate's on-disk repo clones
type Resolver struct {
	// BaseDir mirrors renovate's RENOVATE_BASE_DIR; clones live under BaseDir/repos
	BaseDir string
	// Platform mirrors renovate's RENOVATE_PLATFORM and is a path segment in the clone layout
	Platform string
}

func NewFromEnv() *Resolver {
	// Mirror renovate's baseDir resolution (lib/workers/global/initialize.ts):
	// baseDir = RENOVATE_BASE_DIR, else (RENOVATE_TMPDIR || os tmp) + "/renovate".
	baseDir := os.Getenv("RENOVATE_BASE_DIR")
	if baseDir == "" {
		tmp := os.Getenv("RENOVATE_TMPDIR")
		if tmp == "" {
			tmp = os.TempDir()
		}
		baseDir = filepath.Join(tmp, "renovate")
	}
	return &Resolver{
		BaseDir:  baseDir,
		Platform: os.Getenv("RENOVATE_PLATFORM"),
	}
}

func (r *Resolver) Resolve(repo string) (team, name string) {
	slug, _, _ := strings.Cut(repo, "?") // drop per-repo options such as "?loglevel=debug"
	team, name = Unknown, slug

	path := filepath.Join(r.BaseDir, "repos", r.Platform, filepath.FromSlash(slug), "config.json")
	data, err := os.ReadFile(path) // #nosec G304 -- path built from trusted env + discovered repo slug
	if err != nil {
		logrus.Warnf("repoowner: cannot read %s, defaulting team=%q: %s", path, Unknown, err)
		return team, name
	}

	var config repoConfig
	if err := json.Unmarshal(data, &config); err != nil {
		logrus.Warnf("repoowner: cannot parse %s, defaulting team=%q: %s", path, Unknown, err)
		return team, name
	}

	if config.Team != "" {
		team = config.Team
	}
	if config.ServiceName != "" {
		name = config.ServiceName
	}
	return team, name
}
