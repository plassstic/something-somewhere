package service

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"plassstic.tech/trainee/avito/internal/repo"
	"plassstic.tech/trainee/avito/internal/schema"
)

var _ Service = service{}

type service struct {
	pool *pgxpool.Pool
}

func decide(ctx context.Context, tx pgx.Tx, err *schema.Err) {
	if err != nil {
		e := tx.Rollback(ctx)
		log.Debug().Any("e", e).Msg("rollback")
	} else {
		e := tx.Commit(ctx)
		log.Debug().Any("e", e).Msg("commit")
	}
}

func rb(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func (s service) begin(ctx context.Context) (pgx.Tx, *schema.Err) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, schema.Err{}.Wrap(schema.Unknown, err)
	}
	return tx, nil
}

type Service interface {
	AddTeam(ctx context.Context, team schema.Team) (*schema.Team, *schema.Err)
	GetTeam(ctx context.Context, teamName string) (*schema.Team, *schema.Err)
	SetUserActive(ctx context.Context, userID string, isActive bool) (*schema.User, *schema.Err)
	CreatePR(ctx context.Context, req schema.CreatePRRequest) (*schema.PullRequest, *schema.Err)
	MergePR(ctx context.Context, prID string) (*schema.PullRequest, *schema.Err)
	ReassignReviewer(ctx context.Context, prID, oldUserID string) (string, *schema.PullRequest, *schema.Err)
	GetUserReviews(ctx context.Context, userID string) ([]schema.PullRequestShort, *schema.Err)
}

func New(pool *pgxpool.Pool) Service {
	return &service{
		pool: pool,
	}
}

func (s service) AddTeam(ctx context.Context, team schema.Team) (t *schema.Team, err *schema.Err) {
	var tx pgx.Tx
	if tx, err = s.begin(ctx); err != nil {
		return
	}

	t, err = repo.R(tx).AddTeamWithMembers(ctx, team)
	decide(ctx, tx, err)

	return
}

func (s service) GetTeam(ctx context.Context, teamName string) (t *schema.Team, err *schema.Err) {
	var tx pgx.Tx
	if tx, err = s.begin(ctx); err != nil {
		return
	}

	t, err = repo.R(tx).GetTeamWithMembers(ctx, teamName)
	decide(ctx, tx, err)
	return
}

func (s service) SetUserActive(ctx context.Context, userID string, isActive bool) (u *schema.User, err *schema.Err) {
	var tx pgx.Tx
	if tx, err = s.begin(ctx); err != nil {
		return
	}

	u, err = repo.R(tx).SetUserActive(ctx, userID, isActive)
	decide(ctx, tx, err)
	return
}

func (s service) CreatePR(ctx context.Context, req schema.CreatePRRequest) (pr *schema.PullRequest, err *schema.Err) {
	var tx pgx.Tx
	if tx, err = s.begin(ctx); err != nil {
		return
	}

	prc := schema.PullReqCreate{
		PRId:     req.PRId,
		Name:     req.Name,
		AuthorID: req.AuthorID,
	}

	if pr, err = repo.R(tx).CreatePR(ctx, prc); err != nil {
		rb(ctx, tx)
		return
	}

	var reviewers []string
	reviewers, err = repo.R(tx).AssignReviewersToPR(ctx, req.PRId, req.AuthorID)

	decide(ctx, tx, err)
	pr.AssignedReviewers = reviewers

	return
}

func (s service) MergePR(ctx context.Context, prID string) (pr *schema.PullRequest, err *schema.Err) {
	var tx pgx.Tx
	if tx, err = s.begin(ctx); err != nil {
		return
	}
	pr, err = repo.R(tx).MergePR(ctx, prID)
	decide(ctx, tx, err)
	return
}

func (s service) ReassignReviewer(ctx context.Context, prID, oldUserID string) (newUserID string, updatedPR *schema.PullRequest, err *schema.Err) {
	var tx pgx.Tx
	if tx, err = s.begin(ctx); err != nil {
		return
	}

	newUserID, updatedPR, err = repo.R(tx).ReassignReviewer(ctx, prID, oldUserID)
	decide(ctx, tx, err)
	return
}

func (s service) GetUserReviews(ctx context.Context, userID string) (prs []schema.PullRequestShort, err *schema.Err) {
	var tx pgx.Tx
	if tx, err = s.begin(ctx); err != nil {
		return
	}

	prs, err = repo.R(tx).GetUserReviews(ctx, userID)
	decide(ctx, tx, err)
	return
}
