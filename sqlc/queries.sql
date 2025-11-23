-- name: CreateTeam :one
insert into teams (team_name) 
values ($1)
returning team_name;

-- name: GetTeam :one
select team_name from teams 
where team_name = $1;

-- name: GetUsersForTeam :many
select u.user_id, u.user_name, u.is_active
from users_to_teams ut
inner join users u using (user_id)
where ut.team_name = $1;

-- name: EnsureUsers :batchexec
insert into users (user_id, user_name, is_active) 
values ($1, $2, $3) 
on conflict (user_id) do update
set user_name = excluded.user_name,
    is_active = excluded.is_active;

-- name: AddUsersToTeam :batchexec
insert into users_to_teams (user_id, team_name) 
values ($1, $2) 
on conflict (user_id, team_name) do nothing;

-- name: UserSetIsActive :one
update users
set is_active = $2
where user_id = $1
returning *;

-- name: GetUserWithTeam :one
select u.user_id, u.user_name, u.is_active, ut.team_name
from users u
left join users_to_teams ut using (user_id)
where u.user_id = $1;

-- name: GetUserTeam :one
select team_name
from users_to_teams
where user_id = $1;

-- name: CreatePR :one
insert into pull_requests (pull_req_id, pull_req_name, author_id)
values ($1, $2, $3)
returning *;

-- name: GetPR :one
select * from pull_requests
where pull_req_id = $1;

-- name: MergePR :one
update pull_requests
set pull_req_status = 'merged'::prstat
where pull_req_id = $1
returning *;

-- name: AddReviewer :one
insert into reviewers_to_pull_requests (user_id, pull_req_id)
values ($1, $2)
returning user_id;

-- name: GetReviewersForPR :many
select user_id
from reviewers_to_pull_requests
where pull_req_id = $1;

-- name: RemoveReviewer :exec
delete from reviewers_to_pull_requests
where pull_req_id = $1 and user_id = $2;

-- name: GetPRsReviewedByUser :many
select prq.pull_req_id, prq.pull_req_name, prq.author_id, prq.pull_req_status
from pull_requests prq
inner join reviewers_to_pull_requests rtp on rtp.pull_req_id = prq.pull_req_id
where rtp.user_id = $1;

-- name: GetUserCoworkers :many
select utt.user_id 
from users_to_teams utt
inner join users_to_teams uttf on utt.team_name = uttf.team_name
where uttf.user_id = $1 
  and utt.user_id <> $1;

-- name: GetActiveCoworkersExcludingUsers :many
select utt.user_id 
from users_to_teams utt
inner join users u on u.user_id = utt.user_id
where utt.team_name = $1
  and u.is_active = true
  and u.user_id <> all($2::text[]);

-- name: CountReviewersForPR :one
select count(*) as reviewer_count
from reviewers_to_pull_requests
where pull_req_id = $1;

-- name: IsReviewerAssigned :one
select exists(
    select 1 from reviewers_to_pull_requests
    where pull_req_id = $1 and user_id = $2
) as is_assigned;

-- name: CheckTeamExists :one
select exists(select 1 from teams where team_name = $1) as exists;

-- name: CheckPRExists :one
select exists(select 1 from pull_requests where pull_req_id = $1) as exists;

-- name: CheckUserExists :one
select exists(select 1 from users where user_id = $1) as exists;

-- name: GetPRwithReviewers :one
select 
    pr.pull_req_id,
    pr.pull_req_name,
    pr.author_id,
    pr.pull_req_status,
    pr.created_at,
    pr.merged_at,
    coalesce(
        array_agg(rtp.user_id) filter (where rtp.user_id is not null),
        array[]::text[]
    ) as assigned_reviewers
from pull_requests pr
left join reviewers_to_pull_requests rtp on pr.pull_req_id = rtp.pull_req_id
where pr.pull_req_id = $1
group by pr.pull_req_id, pr.pull_req_name, pr.author_id, pr.pull_req_status, pr.created_at, pr.merged_at;
