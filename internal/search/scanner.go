package search

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
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
	historyRoots, normalizedProjectDir, projectDirVariants, exists, err := scanner.resolveHistoryRoots(projectDir)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []session.Candidate{}, nil
	}

	type workItem struct {
		path        string
		historyRoot string
	}

	type result struct {
		candidate   session.Candidate
		historyRoot string
		err         error
	}

	files := make([]workItem, 0, 64)
	for _, historyRoot := range historyRoots {
		rootFiles, walkErr := walkJSONLFiles(historyRoot)
		if walkErr != nil {
			return nil, walkErr
		}

		for _, path := range rootFiles {
			files = append(files, workItem{path: path, historyRoot: historyRoot})
		}
	}

	fileCh := make(chan workItem)
	resultCh := make(chan result, scanner.workers)

	var group sync.WaitGroup
	for range scanner.workers {
		group.Add(1)
		go func() {
			defer group.Done()

			for item := range fileCh {
				select {
				case <-ctx.Done():
					resultCh <- result{err: ctx.Err()}
					return
				default:
				}

				candidate, matched, scanErr := scanFile(item.path, query)
				if scanErr != nil {
					resultCh <- result{err: scanErr}
					continue
				}
				candidate.ProjectPath = normalizedProjectDir

				if matched {
					resultCh <- result{candidate: candidate, historyRoot: item.historyRoot}
				}
			}
		}()
	}

	go func() {
		defer close(fileCh)
		for _, item := range files {
			fileCh <- item
		}
	}()

	go func() {
		group.Wait()
		close(resultCh)
	}()

	var scannedCandidates []scannedCandidate
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

		scannedCandidates = append(scannedCandidates, scannedCandidate{
			candidate:   resultItem.candidate,
			historyRoot: resultItem.historyRoot,
		})
	}

	candidates := filterCandidatesForProject(scannedCandidates, normalizedProjectDir, projectDirVariants, len(historyRoots))

	sort.Slice(candidates, func(left int, right int) bool {
		if candidates[left].UpdatedAt.Equal(candidates[right].UpdatedAt) {
			return candidates[left].TranscriptPath < candidates[right].TranscriptPath
		}

		return candidates[left].UpdatedAt.After(candidates[right].UpdatedAt)
	})

	return candidates, nil
}

type scannedCandidate struct {
	candidate   session.Candidate
	historyRoot string
}

type rootMatchState struct {
	matched     bool
	conflicting bool
}

func filterCandidatesForProject(scannedCandidates []scannedCandidate, normalizedProjectDir string, projectDirVariants []string, historyRootCount int) []session.Candidate {
	projectDirSet := make(map[string]struct{}, len(projectDirVariants))
	for _, variant := range projectDirVariants {
		projectDirSet[variant] = struct{}{}
	}

	type resolvedCWD struct {
		normalized string
		ok         bool
	}
	cwdCache := make([]resolvedCWD, len(scannedCandidates))
	for i, item := range scannedCandidates {
		cwdCache[i].normalized, cwdCache[i].ok = normalizeCandidateCWD(item.candidate.CWD)
	}

	rootStates := make(map[string]rootMatchState)
	hasMatchedRoot := false
	for i, item := range scannedCandidates {
		state := rootStates[item.historyRoot]
		if !cwdCache[i].ok {
			rootStates[item.historyRoot] = state
			continue
		}

		if projectDirMatches(cwdCache[i].normalized, projectDirSet) {
			state.matched = true
			hasMatchedRoot = true
		} else {
			state.conflicting = true
		}
		rootStates[item.historyRoot] = state
	}

	dedupedCandidates := make(map[string]session.Candidate, len(scannedCandidates))
	for i, item := range scannedCandidates {
		candidate := item.candidate

		keep := false
		if cwdCache[i].ok {
			keep = projectDirMatches(cwdCache[i].normalized, projectDirSet)
		} else {
			state := rootStates[item.historyRoot]
			keep = historyRootCount == 1 || (!state.conflicting && (state.matched || hasMatchedRoot))
		}
		if !keep {
			continue
		}

		existing, ok := dedupedCandidates[candidate.SessionID]
		if !ok || preferCandidate(candidate, existing) {
			dedupedCandidates[candidate.SessionID] = candidate
		}
	}

	candidates := make([]session.Candidate, 0, len(dedupedCandidates))
	for _, candidate := range dedupedCandidates {
		candidates = append(candidates, candidate)
	}

	return candidates
}

func normalizeCandidateCWD(cwd string) (string, bool) {
	if strings.TrimSpace(cwd) == "" {
		return "", false
	}

	normalizedCWD, err := normalizeProjectDir(cwd)
	if err != nil {
		return "", false
	}

	return normalizedCWD, true
}

func projectDirMatches(candidateCWD string, projectDirSet map[string]struct{}) bool {
	_, ok := projectDirSet[candidateCWD]
	return ok
}

func preferCandidate(candidate session.Candidate, existing session.Candidate) bool {
	if candidate.CanResume != existing.CanResume {
		return candidate.CanResume
	}
	if candidate.HitCount != existing.HitCount {
		return candidate.HitCount > existing.HitCount
	}
	if !candidate.UpdatedAt.Equal(existing.UpdatedAt) {
		return candidate.UpdatedAt.After(existing.UpdatedAt)
	}
	if candidate.TranscriptPath != existing.TranscriptPath {
		return candidate.TranscriptPath < existing.TranscriptPath
	}

	return filepath.Clean(candidate.CWD) < filepath.Clean(existing.CWD)
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
