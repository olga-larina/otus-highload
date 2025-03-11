-- +goose Up
-- +goose StatementBegin
alter table users add column user_status int not null default 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table users drop column user_status;
-- +goose StatementEnd