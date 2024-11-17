# Собираем в гошке
FROM golang:1.23.1-alpine AS build

# Устанавливаем нужные зависимости для статической компиляции
RUN apk add --no-cache git build-base

# Устанавливаем goose
RUN go install -ldflags '-extldflags "-static"' github.com/pressly/goose/v3/cmd/goose@v3.19.2

# На выходе тонкий образ
FROM alpine:3.9

LABEL ORGANIZATION="OTUS Online Education"
LABEL SERVICE="backend-migrator"
LABEL MAINTAINERS="student@otus.ru"

# Копируем скомпилированный статический бинарник
COPY --from=build /go/bin/goose /app/goose

ENV MIGRATIONS_DIR /app/migrations

CMD /app/goose -dir ${MIGRATIONS_DIR} ${DB_TYPE} "${DB_URI}" up
