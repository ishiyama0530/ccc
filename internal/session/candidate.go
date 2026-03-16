package session

import "time"

const DefaultTitle = "no title"

type Candidate struct {
	SessionID      string
	TranscriptPath string
	CWD            string
	UpdatedAt      time.Time
	HitCount       int
	Preview        string
	SearchQuery    string
	Title          string
	ProjectPath    string
	CanResume      bool
}
