package renovate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fortnoxab/renovator/pkg/command"
	"github.com/sirupsen/logrus"
)

type Runner struct {
	// BaseDir and Platform mirror renovate's, so we can find the repo clones it makes
	BaseDir   string
	Platform  string
	commander command.Commander
}

func NewRunner(c command.Commander) *Runner {
	return &Runner{
		BaseDir:   baseDirFromEnv(),
		Platform:  os.Getenv("RENOVATE_PLATFORM"),
		commander: c,
	}
}

// baseDirFromEnv mirrors renovate's own resolution in lib/workers/global/initialize.ts.
func baseDirFromEnv() string {
	dir := os.Getenv("RENOVATE_BASE_DIR")
	if dir == "" {
		tmp := os.Getenv("RENOVATE_TMPDIR")
		if tmp == "" {
			tmp = os.TempDir()
		}
		dir = filepath.Join(tmp, "renovate")
	}
	return dir
}

// Run is a finished renovate run. Cleanup is never nil.
type Run struct {
	Slug     string
	CloneDir string
	Cleanup  func()
}

// RunRenovate keeps the clone via persistRepoData so the caller can read from CloneDir,
// and hands back Cleanup to remove it again.
func (r *Runner) RunRenovate(repo string) (Run, error) {
	slug, options, _ := strings.Cut(repo, "?")

	dir := filepath.Join(r.BaseDir, "repos", r.Platform, filepath.FromSlash(slug))
	run := Run{
		Slug:     slug,
		CloneDir: dir,
		Cleanup: func() {
			if err := os.RemoveAll(dir); err != nil {
				logrus.Warnf("error removing clone: %s, err: %s", dir, err)
			}
		},
	}

	env := []string{}
	switch options {
	case "loglevel=debug":
		env = []string{"LOG_LEVEL=debug"}
	}
	err := r.commander.RunWithEnv(env, "renovate", "--persist-repo-data=true", slug)
	if err != nil {
		return run, fmt.Errorf("error running renovate on repo: %s, err: %w", slug, err)
	}
	return run, nil
}

// DoAutoDiscover returns a list of repos
func (r *Runner) DoAutoDiscover() ([]string, error) {

	file, err := createTempFile()
	if err != nil {
		return nil, fmt.Errorf("error creating tempfile, err: %w", err)
	}
	defer os.Remove(file.Name())

	err = r.commander.Run("renovate", "--write-discovered-repos", file.Name())
	if err != nil {
		return nil, fmt.Errorf("error running renovate discovery, err: %w", err)
	}

	fileData, err := os.ReadFile(file.Name())
	if err != nil {
		return nil, fmt.Errorf("error reading repolist file, err: %w", err)
	}

	repos := []string{}

	err = json.Unmarshal(fileData, &repos)
	if err != nil {
		return nil, fmt.Errorf("error unmarshling repolist, err: %w", err)
	}

	return repos, nil

}

func createTempFile() (*os.File, error) {
	file, err := os.CreateTemp("", "renovator_")
	if err != nil {
		return nil, err
	}

	err = file.Close()
	if err != nil {
		return nil, err
	}
	return file, nil
}
