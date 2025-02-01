-- +goose Up
-- +goose StatementBegin
create index friends_friend_id_idx on friends(friend_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index if exists friends_friend_id_idx;
-- +goose StatementEnd