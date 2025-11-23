package repo

import (
	"context"
	"fmt"
	"math/rand"
	"slices"

	"github.com/jackc/pgx/v5"
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
	b, lerr := r.qs.IsReviewerAssigned(ctx, gensql.IsReviewerAssignedParams{
		PullReqID: prID,
		UserID:    oldUserID,
	})

	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	} else if !b {
		err = schema.Err{}.Wrap(schema.NotAssigned, fmt.Errorf("user %s is not assigned to PR %s", oldUserID, prID))
		return
	}

	prRow, lerr := r.qs.GetPRwithReviewers(ctx, prID)

	if lerr != nil {
		err = schema.Err{}.Wrap(schema.NotFound, fmt.Errorf("PR %s not found", prID))
		return
	}
	if prRow.PullReqStatus == gensql.PrstatMerged {
		err = schema.Err{}.Wrap(schema.PRMerged, fmt.Errorf("cannot reassign on merged PR"))
		return
	}

	teamName, lerr := r.qs.GetUserTeam(ctx, oldUserID)

	if lerr != nil {
		err = schema.Err{}.Wrap(schema.NotFound, fmt.Errorf("user %s not found", oldUserID))
		return
	}

	candidates, lerr := r.qs.GetActiveCoworkersExcludingUsers(ctx, gensql.GetActiveCoworkersExcludingUsersParams{
		TeamName: teamName,
		Column2:  slices.Concat([]string{prRow.AuthorID}, schema.StrSlice(prRow.AssignedReviewers)),
	})

	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}
	if len(candidates) == 0 {
		err = schema.Err{}.Wrap(schema.NoCandidate, fmt.Errorf("no active replacement candidate in team"))
		return
	}

	newUserID = candidates[rand.Intn(len(candidates))]

	lerr = r.qs.RemoveReviewer(ctx, gensql.RemoveReviewerParams{
		PullReqID: prID,
		UserID:    oldUserID,
	})

	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

	_, lerr = r.qs.AddReviewer(ctx, gensql.AddReviewerParams{
		UserID:    newUserID,
		PullReqID: prID,
	})

	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

	updatedPR = schema.PullRequest{}.FromRowWithRevs(prRow)
	updatedPR.AssignedReviewers = lo.Replace(updatedPR.AssignedReviewers, oldUserID, newUserID, 1)

	return
}

func (r repository) AssignReviewersToPR(ctx context.Context, prID, authorID string) (reviewers []string, err *schema.Err) {
	teamName, lerr := r.qs.GetUserTeam(ctx, authorID)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.NotFound, fmt.Errorf("user %s not found", authorID))
		return
	}

	candidates, lerr := r.qs.GetActiveCoworkersExcludingUsers(ctx, gensql.GetActiveCoworkersExcludingUsersParams{
		TeamName: teamName,
		Column2:  []string{authorID},
	})
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

	assignCount := 2
	if len(candidates) < 2 {
		assignCount = len(candidates)
	}

	reviewers = candidates[:assignCount]

	lerr = r.AddReviewersToPR(ctx, prID, reviewers)
	if lerr != nil {
		err = schema.Err{}.Wrap(schema.Unknown, lerr)
		return
	}

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
