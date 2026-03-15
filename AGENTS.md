# ccc Agent Guide

## Purpose
- `ccc` is a speed-first Go CLI for searching Claude Code transcript `.jsonl` files and resuming the right session quickly.
- The strictest product constraint is stdout purity for the single-match case: only `claude --resume <session_id>` may be printed.
- The default lookup target is the Claude history directory that corresponds to the current working directory. `-d/--dir` swaps in another working directory, not another raw transcript root.

## Architecture Map
- `cmd/ccc`: CLI entrypoint.
- `internal/app`: argument parsing and `0/1/many` orchestration.
- `internal/search`: recursive walk, parallel scan, preview generation.
- `internal/session`: line-oriented transcript extraction and noise filtering.
- `internal/tui`: minimal Bubble Tea picker.
- `internal/resume`: `claude --resume` execution in transcript `cwd`.

## Speed Rules
- Full scan only. Do not add an index unless the task explicitly asks for one.
- Prefer line-by-line parsing over full JSON decode.
- Keep allocations low in the scan path. Avoid copying transcript text unless needed for a match or preview.
- Exclude obvious noise early: `progress`, snapshots, `tool_result`, `tool_use`, `thinking`, and unknown-role content.
- Preserve session-level aggregation: one transcript file equals one candidate.

## Output Contract
- `0` matches: stderr only, non-zero exit, stdout empty.
- `1` match: stdout exactly `claude --resume <session_id>\n`, no TUI, no extra stderr output.
- `2+` matches: TUI only on TTY, otherwise fail on stderr and non-zero.
- `session_id` always comes from the transcript filename, never from transcript JSON fields.

## TDD Workflow
1. Write or extend a failing test first.
2. Implement the smallest change that makes that slice pass.
3. Refactor only after the slice is green.
4. Rerun the narrow package test before moving to the next slice.
5. After a feature area is complete, rerun the full test suite.

## Verification Gates
- `make build`
- `make test`
- `make lint`
- Run focused benchmarks when touching the scan path.
- Do a fresh-context review pass before considering the work done.

## Editing Guidance
- Keep `search`, `session`, `tui`, and `resume` separated so indexing can be added later without untangling behavior.
- Prefer standard library APIs unless a third-party library clearly improves speed or reliability.
- Keep TUI intentionally plain. No extra decoration that slows rendering or obscures the list.
- When changing output behavior, add or update a test that locks stdout/stderr exactly.

## Release Workflow
- Homebrew is the only supported distribution channel.
- Use `make release VERSION=vX.Y.Z TAP_REPO=owner/homebrew-tap`.
- The command tags and pushes the release tag, then runs GoReleaser to publish GitHub release assets and update the Homebrew tap.
- `GITHUB_TOKEN` must be set before running the release target.

## Acceptance Checklist
- Single result prints a clean resume command and nothing else to stdout.
- Multiple results can be selected with `j/k`, arrows, and `Enter`.
- Resume execution uses transcript `cwd`.
- Broken transcript lines do not abort the overall scan.
- README and release instructions stay aligned with the current command behavior.
