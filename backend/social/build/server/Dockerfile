# Собираем в гошке
FROM golang:1.23.1 AS build

ENV BIN_FILE /opt/backend/server
ENV PKG_DIR /go/src/pkg
ENV CODE_DIR /go/src/social

# Контекст сборки - родительский каталог
WORKDIR ${PKG_DIR}
COPY ./pkg/. .
RUN go mod tidy

WORKDIR ${CODE_DIR}
COPY ./social/go.mod .
RUN go mod tidy

COPY ./social/. ${CODE_DIR}

# Собираем статический бинарник Go (без зависимостей на Си API),
# иначе он не будет работать в alpine образе.
ARG LDFLAGS
RUN CGO_ENABLED=0 go build \
        -ldflags "$LDFLAGS" \
        -o ${BIN_FILE} cmd/server/*

# На выходе тонкий образ
FROM alpine:3.9

LABEL ORGANIZATION="OTUS Online Education"
LABEL SERVICE="backend-social-server"
LABEL MAINTAINERS="student@otus.ru"

# Для использования временных зон
RUN apk add --no-cache tzdata

ENV BIN_FILE /opt/backend/server
COPY --from=build ${BIN_FILE} ${BIN_FILE}

ENV CONFIG_FILE /etc/backend/server/config.yaml

CMD ${BIN_FILE} -config ${CONFIG_FILE}
