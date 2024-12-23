-- +goose Up
-- +goose StatementBegin
create table if not exists messages (
    dialog_id       varchar(32),
    message_id      varchar(36),
    content         text,
    from_user_id    varchar(36) not null,
    to_user_id      varchar(36) not null,
    send_time       timestamp not null default now(),
	constraint messages_pk primary key (dialog_id, message_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists messages;
-- +goose StatementEnd