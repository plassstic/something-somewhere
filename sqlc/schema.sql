create type prstat as enum ('open', 'merged');

create table teams
(
    team_name text primary key
);

create table users
(
    user_id   text primary key,
    user_name text not null,
    is_active bool not null default true
);

create table users_to_teams
(
    user_id   text references users on update restrict on delete cascade not null,
    team_name text references teams on update restrict on delete cascade not null,
    primary key (user_id, team_name),
    unique (user_id)
);

create table pull_requests
(
    pull_req_id     text primary key,
    pull_req_name   text                                   not null,
    author_id       text references users (user_id) on update restrict on delete cascade not null,
    pull_req_status prstat default 'open'::prstat not null,

    created_at      timestamp default now() not null,
    merged_at       timestamp
);

create table reviewers_to_pull_requests
(
    user_id     text references users on update restrict on delete cascade,
    pull_req_id text references pull_requests on update restrict on delete cascade,
    primary key (user_id, pull_req_id)
);

create function reviewersconstr()
    returns trigger as
$$
begin
    if (select count(*) from reviewers_to_pull_requests where pull_req_id = new.pull_req_id) = 2 then
        raise exception 'reviewers count for pull request % already eq to 2', new.pull_req_id;
    end if;
    return new;
end;
$$ language plpgsql;

create function validatestatus()
    returns trigger as
$$
declare
    old_status prstat;
begin
    old_status := (select pull_req_status from pull_requests where pull_req_id = new.pull_req_id);
    
    if old_status = 'open'::prstat and new.pull_req_status = 'merged'::prstat then
        new.merged_at := now();
    elsif old_status = 'merged'::prstat and new.pull_req_status = 'open'::prstat then
        raise exception 'pr % already merged', new.pull_req_id;
    end if;
    
    return new;
end;
$$ language plpgsql;

create trigger apply_reviewersconstr
    before insert or update
    on reviewers_to_pull_requests
    for each row
execute function reviewersconstr();

create trigger apply_validatestatus
    before update
    on pull_requests
    for each row
execute function validatestatus();
