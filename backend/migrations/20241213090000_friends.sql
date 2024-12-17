-- +goose Up
-- +goose StatementBegin
create table if not exists friends (
    user_id   varchar(36) not null,
    friend_id varchar(36) not null,
	constraint friends_pk primary key (user_id, friend_id),
    constraint friends_user_id_fk
        foreign key (user_id) 
        references users (id),
    constraint friends_friend_id_fk
        foreign key (friend_id) 
        references users (id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists friends;
-- +goose StatementEnd