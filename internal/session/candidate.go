package session

import "time"

type Candidate struct {
	SessionID      string
	TranscriptPath string
	CWD            string
	UpdatedAt      time.Time
	HitCount       int
	Preview        string
	Title          string
	ProjectPath    string
	CanResume      bool
}
