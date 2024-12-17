-- +goose Up
-- +goose StatementBegin
create index posts_user_id_idx on posts(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index if exists posts_user_id_idx;
-- +goose StatementEnd