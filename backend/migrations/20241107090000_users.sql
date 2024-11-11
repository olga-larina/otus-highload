-- +goose Up
-- +goose StatementBegin
create table if not exists users (
    id              varchar(36),
    first_name      varchar(50) not null,
    second_name     varchar(50) not null,
    city            varchar(50),
    gender          char(1),
    birthdate       date,
    biography       text,
    password_hash   varchar(255) not null,
	constraint users_pk primary key (id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists users;
-- +goose StatementEnd