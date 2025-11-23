package schema

import "fmt"

type ErrorCode string

const (
	PRMerged    ErrorCode = "PR_MERGED"
	TeamExists  ErrorCode = "TEAM_EXISTS"
	PRExists    ErrorCode = "PR_EXISTS"
	NotAssigned ErrorCode = "NOT_ASSIGNED"
	NoCandidate ErrorCode = "NO_CANDIDATE"
	NotFound    ErrorCode = "NOT_FOUND"
	Unknown     ErrorCode = "UNKNOWN"
)

type Err struct {
	Code ErrorCode `json:"code,omitempty"`
	Msg  string    `json:"msg,omitempty"`
}

type ErrorResponse struct {
	Err `json:"error"`
}

func (e Err) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Msg)
}

func (Err) Wrap(code ErrorCode, err error) *Err {
	return &Err{Code: code, Msg: err.Error()}
}
