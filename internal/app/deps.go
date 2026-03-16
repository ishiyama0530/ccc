package app

import (
	"context"
	"os"
	"runtime"

	"github.com/ishiyama0530/ccc/internal/resume"
	"github.com/ishiyama0530/ccc/internal/search"
	"github.com/ishiyama0530/ccc/internal/session"
	"github.com/ishiyama0530/ccc/internal/tui"
	"golang.org/x/term"
)

type Searcher interface {
	Scan(ctx context.Context, root string, query string) ([]session.Candidate, error)
}

type Picker interface {
	Pick(candidates []session.Candidate) (resume.Request, error)
}

type Runner interface {
	Run(ctx context.Context, request resume.Request) error
}

type GetwdFunc func() (string, error)

type IsTTYFunc func() bool

type UpdateNotifier interface {
	Notice(ctx context.Context, currentVersion string) (string, error)
}

func defaultGetwd() (string, error) {
	return os.Getwd()
}

func defaultIsTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func NewService(version string) Service {
	return Service{
		Searcher:       search.NewScanner(runtime.NumCPU()),
		Picker:         tui.Picker{},
		Runner:         resume.Runner{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr},
		Getwd:          defaultGetwd,
		IsTTY:          defaultIsTTY,
		UpdateNotifier: GitHubReleaseUpdateNotifier{},
		Version:        version,
	}
}
