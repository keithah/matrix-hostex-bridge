FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG COMMIT=unknown
ARG TAG=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X 'hostex-matrix-bridge/cmd/mautrix-hostex.Tag=${TAG}' -X 'hostex-matrix-bridge/cmd/mautrix-hostex.Commit=${COMMIT}' -X 'hostex-matrix-bridge/cmd/mautrix-hostex.BuildTime=${BUILD_TIME}'" \
    -o mautrix-hostex .

FROM alpine:3.20

ENV UID=1337 \
    GID=1337 \
    BRIDGEV2=1

RUN apk add --no-cache ffmpeg su-exec ca-certificates bash jq curl yq-go

COPY --from=builder /build/mautrix-hostex /usr/bin/mautrix-hostex
COPY docker-run.sh /docker-run.sh

RUN chmod +x /docker-run.sh

VOLUME /data
WORKDIR /data

EXPOSE 29337

CMD ["/docker-run.sh"]