-- +goose Up
-- +goose StatementBegin
alter table users_to_teams add constraint one_team_per_user UNIQUE (user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
