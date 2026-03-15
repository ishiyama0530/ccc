package resume

import "github.com/ishiyama0530/ccc/internal/session"

type Request struct {
	Candidate session.Candidate
	ExtraArgs []string
}
