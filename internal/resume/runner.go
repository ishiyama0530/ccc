package resume

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
)

type Runner struct {
	ExecPath string
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
	Environ  []string
}

func (runner Runner) Run(ctx context.Context, request Request) error {
	if request.Candidate.CWD == "" {
		return errors.New("cwd is required to resume a session")
	}
	if err := ValidateExtraArgs(request.ExtraArgs); err != nil {
		return err
	}

	execPath := runner.ExecPath
	if execPath == "" {
		execPath = "claude"
	}

	commandArgs := append([]string{}, request.ExtraArgs...)
	commandArgs = append(commandArgs, "--resume", request.Candidate.SessionID)

	command := exec.CommandContext(ctx, execPath, commandArgs...)
	command.Dir = request.Candidate.CWD
	command.Stdin = runner.Stdin
	command.Stdout = runner.Stdout
	command.Stderr = runner.Stderr
	if runner.Environ != nil {
		command.Env = append(os.Environ(), runner.Environ...)
	}

	return command.Run()
}
