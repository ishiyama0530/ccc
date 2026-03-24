package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/ishiyama0530/ccc/internal/session"
	"github.com/ishiyama0530/ccc/internal/tui"
)

type Service struct {
	Searcher Searcher
	Picker   Picker
	Runner   Runner
	Getwd    GetwdFunc
	IsTTY    IsTTYFunc
}

const defaultCandidateLimit = 100

func (service Service) Run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flagSet := flag.NewFlagSet("ccc", flag.ContinueOnError)
	flagSet.SetOutput(stderr)

	searchDir := flagSet.String("d", "", "project directory")
	flagSet.StringVar(searchDir, "dir", "", "project directory")
	limit := flagSet.Int("n", defaultCandidateLimit, "max number of history entries to display")
	flagSet.IntVar(limit, "limit", defaultCandidateLimit, "max number of history entries to display")

	if err := flagSet.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *limit < 1 {
		fmt.Fprintln(stderr, "limit must be at least 1")
		return 2
	}

	query := strings.TrimSpace(strings.Join(flagSet.Args(), " "))

	root := strings.TrimSpace(*searchDir)
	if root == "" {
		getwd := service.Getwd
		if getwd == nil {
			getwd = defaultGetwd
		}

		cwd, err := getwd()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		root = cwd
	}

	if service.Searcher == nil {
		fmt.Fprintln(stderr, "searcher is not configured")
		return 1
	}

	candidates, err := service.Searcher.Scan(ctx, root, query)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	switch len(candidates) {
	case 0:
		fmt.Fprintln(stderr, "no matches found")
		return 1
	}

	candidates = limitCandidates(candidates, *limit)

	isTTY := service.IsTTY
	if isTTY == nil {
		isTTY = func() bool { return false }
	}
	if !isTTY() {
		fmt.Fprintln(stderr, "interactive selection requires a TTY")
		return 1
	}

	if service.Picker == nil || service.Runner == nil {
		fmt.Fprintln(stderr, "picker and runner are required for interactive mode")
		return 1
	}

	request, err := service.Picker.Pick(candidates)
	if err != nil {
		if errors.Is(err, tui.ErrCanceled) {
			return 0
		}
		fmt.Fprintln(stderr, err)
		return 1
	}

	if err := service.Runner.Run(ctx, request); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	return 0
}

func limitCandidates(candidates []session.Candidate, limit int) []session.Candidate {
	if len(candidates) <= limit {
		return candidates
	}

	return candidates[:limit]
}
