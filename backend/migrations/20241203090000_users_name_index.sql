-- +goose Up
-- +goose StatementBegin
create index first_name_second_name_idx on users(first_name text_pattern_ops, second_name text_pattern_ops);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index if exists first_name_second_name_idx;
-- +goose StatementEnd