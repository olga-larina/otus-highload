-- +goose Up
-- +goose StatementBegin
create table if not exists posts (
    id            varchar(36),
    content       text,
    user_id       varchar(36) not null,
    create_time   timestamp not null default now(),
    update_time   timestamp not null default now(),
	constraint posts_pk primary key (id),
    constraint posts_user_id_fk
        foreign key (user_id) 
        references users (id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists posts;
-- +goose StatementEnd