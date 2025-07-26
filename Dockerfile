FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata gcc g++ musl-dev sqlite-dev olm-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG COMMIT=unknown
ARG TAG=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-s -w -X 'main.Tag=${TAG}' -X 'main.Commit=${COMMIT}' -X 'main.BuildTime=${BUILD_TIME}'" \
    -o mautrix-hostex .

FROM alpine:3.20

ENV UID=1337 \
    GID=1337 \
    BRIDGEV2=1

RUN apk add --no-cache ffmpeg su-exec ca-certificates bash jq curl yq-go olm sqlite

COPY --from=builder /build/mautrix-hostex /usr/bin/mautrix-hostex
COPY docker-run.sh /docker-run.sh

RUN chmod +x /docker-run.sh

VOLUME /data
WORKDIR /data

EXPOSE 29337

CMD ["/docker-run.sh"]