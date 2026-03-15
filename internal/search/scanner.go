package search

import (
	"context"
	"errors"
	"os"
	"runtime"
	"sort"
	"sync"

	"github.com/ishiyama0530/ccc/internal/session"
)

type Scanner struct {
	workers      int
	projectsRoot string
}

func NewScanner(workers int) Scanner {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	return Scanner{workers: workers}
}

func (scanner Scanner) Scan(ctx context.Context, projectDir string, query string) ([]session.Candidate, error) {
	historyRoot, normalizedProjectDir, exists, err := scanner.resolveHistoryRoot(projectDir)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []session.Candidate{}, nil
	}

	files, err := walkJSONLFiles(historyRoot)
	if err != nil {
		return nil, err
	}

	type result struct {
		candidate session.Candidate
		err       error
	}

	fileCh := make(chan string)
	resultCh := make(chan result, scanner.workers)

	var group sync.WaitGroup
	for range scanner.workers {
		group.Add(1)
		go func() {
			defer group.Done()

			for path := range fileCh {
				select {
				case <-ctx.Done():
					resultCh <- result{err: ctx.Err()}
					return
				default:
				}

				candidate, matched, scanErr := scanFile(path, query)
				if scanErr != nil {
					resultCh <- result{err: scanErr}
					continue
				}
				candidate.ProjectPath = normalizedProjectDir

				if matched {
					resultCh <- result{candidate: candidate}
				}
			}
		}()
	}

	go func() {
		defer close(fileCh)
		for _, path := range files {
			fileCh <- path
		}
	}()

	go func() {
		group.Wait()
		close(resultCh)
	}()

	var candidates []session.Candidate
	for resultItem := range resultCh {
		if resultItem.err != nil {
			if errors.Is(resultItem.err, context.Canceled) || errors.Is(resultItem.err, context.DeadlineExceeded) {
				return nil, resultItem.err
			}
			continue
		}

		if resultItem.candidate.SessionID == "" {
			continue
		}

		candidates = append(candidates, resultItem.candidate)
	}

	sort.Slice(candidates, func(left int, right int) bool {
		if candidates[left].UpdatedAt.Equal(candidates[right].UpdatedAt) {
			return candidates[left].TranscriptPath < candidates[right].TranscriptPath
		}

		return candidates[left].UpdatedAt.After(candidates[right].UpdatedAt)
	})

	return candidates, nil
}

func scanFile(path string, query string) (session.Candidate, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return session.Candidate{}, false, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return session.Candidate{}, false, err
	}

	candidate, matched, err := session.ExtractCandidate(path, file, query)
	if err != nil {
		return session.Candidate{}, false, err
	}

	candidate.UpdatedAt = info.ModTime()

	return candidate, matched, nil
}
