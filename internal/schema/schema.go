package schema

import (
	"plassstic.tech/trainee/avito/gensql"
)

type TeamMember struct {
	UserID   string `db:"user_id" json:"user_id"`
	UserName string `db:"user_name" json:"username"`
	IsActive bool   `db:"is_active" json:"is_active"`
}

func (TeamMember) FromDDL(user gensql.User) TeamMember {
	return TeamMember{
		UserID:   user.UserID,
		UserName: user.UserName,
		IsActive: user.IsActive,
	}
}

type User struct {
	UserID   string `db:"user_id"`
	UserName string `db:"user_name"`
	TeamName string `db:"team_name"`
	IsActive bool   `db:"is_active"`
}

func (u User) EnsureSchema() gensql.EnsureUsersParams {
	return gensql.EnsureUsersParams{
		UserID:   u.UserID,
		UserName: u.UserName,
		IsActive: u.IsActive,
	}
}

func (u User) AddToTeamSchema(teamName string) gensql.AddUsersToTeamParams {
	return gensql.AddUsersToTeamParams{
		TeamName: teamName,
		UserID:   u.UserID,
	}
}

func (User) FromDDL(user gensql.User) User {
	return User{
		UserID:   user.UserID,
		UserName: user.UserName,
		IsActive: user.IsActive,
	}
}

type Team struct {
	TeamName string       `db:"team_name" json:"team_name"`
	Members  []TeamMember `json:"members"`
}

type PullRequestShort struct {
	PRId     string        `db:"pull_req_id" json:"pull_request_id"`
	Name     string        `db:"pull_req_name" json:"pull_request_name"`
	AuthorId string        `db:"author_id" json:"author_id"`
	Status   gensql.Prstat `json:"status"`
}

func (PullRequestShort) FromDDL(ddl gensql.GetPRsReviewedByUserRow) PullRequestShort {
	return PullRequestShort{
		PRId:     ddl.PullReqID,
		Name:     ddl.PullReqName,
		AuthorId: ddl.AuthorID,
		Status:   ddl.PullReqStatus,
	}
}

func StrSlice(v any) []string {
	if v == nil {
		return nil
	}

	if s, ok := v.([]string); ok {
		return s
	}

	return nil
}

type PullRequest struct {
	PullRequestShort
	AssignedReviewers []string `json:"assigned_reviewers"`
	CreatedAt         string   `db:"created_at" json:"createdAt"`
	MergedAt          string   `db:"merged_at" json:"mergedAt"`
}

func (PullRequest) FromRowWithRevs(ddl gensql.GetPRwithReviewersRow) *PullRequest {
	var mergedAt string
	if ddl.MergedAt.Valid {
		mergedAt = ddl.MergedAt.Time.Format("2006-01-02 15:04:05")
	}
	return &PullRequest{
		PullRequestShort: PullRequestShort{
			PRId:     ddl.PullReqID,
			Name:     ddl.PullReqName,
			AuthorId: ddl.AuthorID,
			Status:   ddl.PullReqStatus,
		},
		AssignedReviewers: StrSlice(ddl.AssignedReviewers),
		CreatedAt:         ddl.CreatedAt.Time.Format("2006-01-02 15:04:05"),
		MergedAt:          mergedAt,
	}
}

func (PullRequest) FromDDL(ddl gensql.PullRequest, revs []string) *PullRequest {
	var mergedAt string
	if ddl.MergedAt.Valid {
		mergedAt = ddl.MergedAt.Time.Format("2006-01-02 15:04:05")
	}
	return &PullRequest{
		PullRequestShort: PullRequestShort{
			PRId:     ddl.PullReqID,
			Name:     ddl.PullReqName,
			AuthorId: ddl.AuthorID,
			Status:   ddl.PullReqStatus,
		},
		CreatedAt:         ddl.CreatedAt.Time.Format("2006-01-02 15:04:05"),
		MergedAt:          mergedAt,
		AssignedReviewers: revs,
	}
}

type PullReqCreate struct {
	PRId     string `json:"pull_request_id"`
	Name     string `json:"pull_request_name"`
	AuthorID string `json:"author_id"`
}

func (prc PullReqCreate) ToCreateParams() gensql.CreatePRParams {
	return gensql.CreatePRParams{
		PullReqID:   prc.PRId,
		PullReqName: prc.Name,
		AuthorID:    prc.AuthorID,
	}
}

type SetUserActiveRequest struct {
	UserID   string `json:"user_id" validate:"required"`
	IsActive bool   `json:"is_active"`
}

type CreatePRRequest struct {
	PRId     string `json:"pull_request_id" validate:"required"`
	Name     string `json:"pull_request_name" validate:"required"`
	AuthorID string `json:"author_id" validate:"required"`
}

type MergePRRequest struct {
	PRId string `json:"pull_request_id" validate:"required"`
}

type ReassignReviewerRequest struct {
	PRId      string `json:"pull_request_id" validate:"required"`
	OldUserID string `json:"old_user_id" validate:"required"`
}

type GetUserReviewsQuery struct {
	UserID string `query:"user_id" validate:"required"`
}

type GetTeamQuery struct {
	TeamName string `query:"team_name" validate:"required"`
}

type AddTeamResponse struct {
	Team Team `json:"team"`
}

type UserResponse struct {
	User User `json:"user"`
}

type PRResponse struct {
	PR PullRequest `json:"pr"`
}

type ReassignResponse struct {
	PR      PullRequest `json:"pr"`
	NewUser string      `json:"replaced_by"`
}

type UserReviewsResponse struct {
	UserID       string             `json:"user_id"`
	PullRequests []PullRequestShort `json:"pull_requests"`
}

type HealthResponse struct {
	Status string `json:"status"`
}
