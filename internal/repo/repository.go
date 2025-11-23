package repo

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"plassstic.tech/trainee/avito/gensql"
	"plassstic.tech/trainee/avito/internal/schema"
)

var _ Repository = repository{}

type repository struct {
	qs *gensql.Queries
}

type Repository interface {
	AddTeamWithMembers(ctx context.Context, team schema.Team) (*schema.Team, *schema.Err)
	GetTeamWithMembers(ctx context.Context, teamName string) (*schema.Team, *schema.Err)
	SetUserActive(ctx context.Context, userID string, isActive bool) (*schema.User, *schema.Err)
	CreatePR(ctx context.Context, pr schema.PullReqCreate) (*schema.PullRequest, *schema.Err)
	MergePR(ctx context.Context, prID string) (*schema.PullRequest, *schema.Err)
	ReassignReviewer(ctx context.Context, prID, oldUserID string) (string, *schema.PullRequest, *schema.Err)
	GetUserReviews(ctx context.Context, userID string) ([]schema.PullRequestShort, *schema.Err)
	GetPR(ctx context.Context, prID string) (*schema.PullRequest, *schema.Err)
	GetReviewersForPR(ctx context.Context, prID string) ([]string, *schema.Err)
	AssignReviewersToPR(ctx context.Context, prID, authorID string) ([]string, *schema.Err)
}

func R(tx pgx.Tx) Repository {
	return &repository{qs: gensql.New(tx)}
}

func (r repository) AddTeam(ctx context.Context, teamName string) (string, error) {
	return r.qs.CreateTeam(ctx, teamName)
}

func (r repository) AddUsersToTeam(ctx context.Context, users []schema.User, teamName string) (err error) {
	r.qs.EnsureUsers(ctx, lo.Map(users, func(user schema.User, _ int) gensql.EnsureUsersParams {
		return user.EnsureSchema()
	})).Exec(
		func(_ int, ierr error) {
			if ierr != nil {
				err = ierr
				return
			}
		},
	)
	if err != nil {
		return
	}

	r.qs.AddUsersToTeam(ctx, lo.Map(users, func(user schema.User, _ int) gensql.AddUsersToTeamParams {
		return user.AddToTeamSchema(teamName)
	})).Exec(
		func(_ int, ierr error) {
			if ierr != nil {
				err = ierr
				return
			}
		},
	)
	if err != nil {
		return
	}

	return
}

func (r repository) GetTeamWithMembers(ctx context.Context, teamName string) (team *schema.Team, err *schema.Err) {
	b, lerr := r.qs.CheckTeamExists(ctx, teamName)
	if !b {
		err = schema.Err{}.Wrap(schema.NotFound, fmt.Errorf("team does not exist"))
		return
	} else if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

	mbs, lerr := r.qs.GetUsersForTeam(ctx, teamName)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

	team = &schema.Team{
		TeamName: teamName,
		Members: lo.Map(mbs, func(user gensql.User, _ int) schema.TeamMember {
			return schema.TeamMember{}.FromDDL(user)
		}),
	}

	return
}

func (r repository) AddTeamWithMembers(ctx context.Context, team schema.Team) (res *schema.Team, err *schema.Err) {
	b, lerr := r.qs.CheckTeamExists(ctx, team.TeamName)

	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	} else if b {
		err = schema.Err{}.Wrap(schema.TeamExists, fmt.Errorf("team %s already exists", team.TeamName))
		return
	}

	_, lerr = r.qs.CreateTeam(ctx, team.TeamName)

	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

	var users []schema.User
	for _, member := range team.Members {
		users = append(users, schema.User{
			UserID:   member.UserID,
			UserName: member.UserName,
			IsActive: member.IsActive,
		})
	}

	lerr = r.AddUsersToTeam(ctx, users, team.TeamName)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

	res = &team
	return
}

func (r repository) SetUserActive(ctx context.Context, userID string, isActive bool) (user *schema.User, err *schema.Err) {
	u, lerr := r.qs.UserSetIsActive(ctx, gensql.UserSetIsActiveParams{
		UserID:   userID,
		IsActive: isActive,
	})
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.NotFound, fmt.Errorf("user %s not found", userID))
		return
	}

	user = &schema.User{}
	user.UserID = u.UserID
	user.UserName = u.UserName
	user.IsActive = u.IsActive

	team, lerr := r.qs.GetUserTeam(ctx, userID)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.NotFound, fmt.Errorf("user %s not found", userID))
		return
	}
	user.TeamName = team

	return
}

func (r repository) CreatePR(ctx context.Context, prc schema.PullReqCreate) (res *schema.PullRequest, err *schema.Err) {
	b, lerr := r.qs.CheckPRExists(ctx, prc.PRId)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	} else if b {
		err = schema.Err{}.Wrap(schema.PRExists, fmt.Errorf("PR %s already exists", prc.PRId))
		return
	}

	b, lerr = r.qs.CheckUserExists(ctx, prc.AuthorID)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	} else if !b {
		err = schema.Err{}.Wrap(schema.NotFound, fmt.Errorf("user %s not found", prc.AuthorID))
		return
	}

	_, lerr = r.qs.CreatePR(ctx, prc.ToCreateParams())
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

	pr, lerr := r.qs.GetPR(ctx, prc.PRId)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

	res = schema.PullRequest{}.FromDDL(pr, nil)
	return
}

func (r repository) MergePR(ctx context.Context, prID string) (res *schema.PullRequest, err *schema.Err) {
	b, lerr := r.qs.CheckPRExists(ctx, prID)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	} else if !b {
		err = schema.Err{}.Wrap(schema.NotFound, fmt.Errorf("PR %s not found", prID))
		return
	}

	merged, lerr := r.qs.MergePR(ctx, prID)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

	reviewers, lerr := r.qs.GetReviewersForPR(ctx, prID)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

	res = schema.PullRequest{}.FromDDL(merged, reviewers)

	return
}

func (r repository) GetUserReviews(ctx context.Context, userID string) (prs []schema.PullRequestShort, err *schema.Err) {
	b, lerr := r.qs.CheckUserExists(ctx, userID)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	} else if !b {
		err = schema.Err{}.Wrap(schema.NotFound, fmt.Errorf("user %s not found", userID))
		return
	}

	prsDDL, lerr := r.qs.GetPRsReviewedByUser(ctx, userID)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

	prs = lo.Map(prsDDL, func(pr gensql.GetPRsReviewedByUserRow, _ int) schema.PullRequestShort {
		return schema.PullRequestShort{}.FromDDL(pr)
	})

	return
}

func (r repository) AddReviewersToPR(ctx context.Context, prID string, reviewers []string) error {
	for _, userID := range reviewers {
		if _, err := r.qs.AddReviewer(ctx, gensql.AddReviewerParams{
			UserID:    userID,
			PullReqID: prID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r repository) ReassignReviewer(ctx context.Context, prID, oldUserID string) (newUserID string, updatedPR *schema.PullRequest, err *schema.Err) {
	var prRow gensql.GetPRwithReviewersRow
	var teamName string
	var candidates []string

	if err = r.isReviewerAssigned(ctx, prID, oldUserID); err != nil {
		return
	}

	if prRow, err = r.getPRWithReviewers(ctx, prID); err != nil {
		return
	}

	if prRow.PullReqStatus == gensql.PrstatMerged {
		err = schema.Err{}.Wrap(schema.PRMerged, fmt.Errorf("cannot reassign on merged PR"))
		return
	}

	if teamName, err = r.getUserTeam(ctx, oldUserID); err != nil {
		return
	}

	exclude := []string{prRow.AuthorID}
	if cst, ok := prRow.AssignedReviewers.([]any); ok && len(cst) > 0 {
		for _, r := range cst {
			if sr, ok := r.(string); ok {
				exclude = append(exclude, sr)
			}
		}
	}

	if candidates, err = r.getActiveTeammates(ctx, teamName, exclude); err != nil {
		return
	}

	if len(candidates) == 0 {
		err = schema.Err{}.Wrap(schema.NoCandidate, fmt.Errorf("no active replacement candidate in team"))
		return
	}

	newUserID = candidates[rand.Intn(len(candidates))]

	if lerr := r.qs.RemoveReviewer(ctx, gensql.RemoveReviewerParams{
		PullReqID: prID,
		UserID:    oldUserID,
	}); lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

	if _, lerr := r.qs.AddReviewer(ctx, gensql.AddReviewerParams{
		UserID:    newUserID,
		PullReqID: prID,
	}); lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

	if prRow, err = r.getPRWithReviewers(ctx, prID); err != nil {
		return
	}

	updatedPR = schema.PullRequest{}.FromRowWithRevs(prRow)

	return
}

func (r repository) getUserTeam(ctx context.Context, userID string) (teamName string, err *schema.Err) {
	var lerr error
	if teamName, lerr = r.qs.GetUserTeam(ctx, userID); lerr != nil {
		err = schema.Err{}.Wrap(schema.NotFound, fmt.Errorf("user %s not found", userID))
	}

	log.Debug().
		Any("userid", userID).
		Any("teamName", teamName).
		AnErr("err", err).
		Msg("getUserTeam")

	return
}

func (r repository) getPRWithReviewers(ctx context.Context, prID string) (pr gensql.GetPRwithReviewersRow, err *schema.Err) {
	var lerr error
	if pr, lerr = r.qs.GetPRwithReviewers(ctx, prID); lerr != nil {
		err = schema.Err{}.Wrap(schema.NotFound, fmt.Errorf("PR %s not found", prID))
	}

	log.Debug().
		Any("prid", prID).
		Any("pr", pr).
		AnErr("err", err).
		Msg("getPRWithReviewers")

	return
}

func (r repository) isReviewerAssigned(ctx context.Context, prID string, userID string) (err *schema.Err) {
	b, lerr := r.qs.IsReviewerAssigned(ctx, gensql.IsReviewerAssignedParams{
		PullReqID: prID,
		UserID:    userID,
	})

	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
	} else if !b {
		err = schema.Err{}.Wrap(schema.NotAssigned, fmt.Errorf("user %s is not assigned to PR %s", userID, prID))
	}

	log.Debug().
		Any("userid", userID).
		Any("prID", prID).
		AnErr("err", err).
		Msg("isReviewerAssigned")

	return
}

func (r repository) AssignReviewersToPR(ctx context.Context, prID, authorID string) (reviewers []string, err *schema.Err) {
	var candidates []string
	teamName, lerr := r.qs.GetUserTeam(ctx, authorID)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.NotFound, fmt.Errorf("user %s not found", authorID))
		return
	}

	if candidates, err = r.getActiveTeammates(ctx, teamName, []string{authorID}); err != nil {
		return
	}

	reviewers = candidates[:min(len(candidates), 2)]

	if lerr = r.AddReviewersToPR(ctx, prID, reviewers); lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

	return
}

func (r repository) getActiveTeammates(ctx context.Context, teamName string, exclude []string) (candidates []string, err *schema.Err) {
	var lerr error

	if candidates, lerr = r.qs.GetActiveTeammates(ctx, gensql.GetActiveTeammatesParams{
		TeamName: teamName,
		Column2:  exclude,
	}); lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
	}

	log.Debug().
		Any("team", teamName).
		Any("exc", exclude).
		Any("candidates", candidates).
		AnErr("err", err).
		Msg("getActiveTeammates")

	return
}

func (r repository) GetPR(ctx context.Context, prID string) (pr *schema.PullRequest, err *schema.Err) {
	prDDL, lerr := r.qs.GetPR(ctx, prID)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.NotFound, fmt.Errorf("PR %s not found", prID))
		return
	}

	reviewers, lerr := r.qs.GetReviewersForPR(ctx, prID)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

	pr = schema.PullRequest{}.FromDDL(prDDL, reviewers)
	return
}

func (r repository) GetReviewersForPR(ctx context.Context, prID string) (reviewers []string, err *schema.Err) {
	reviewersDDL, lerr := r.qs.GetReviewersForPR(ctx, prID)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}
	reviewers = reviewersDDL
	return
}
