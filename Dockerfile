# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.27-alpine AS build

WORKDIR /src

# Dependency layer: cached as long as go.mod/go.sum don't change.
COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY core ./core

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/arietta-bot .

# ---- runtime stage ----
FROM alpine:3.22

# ca-certificates is required for the outbound HTTPS calls to api.github.com.
RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 arietta \
    && adduser -S -D -H -u 10001 -G arietta arietta

WORKDIR /app

COPY --from=build /out/arietta-bot /usr/local/bin/arietta-bot

# Point GITHUB_PRIVATE_KEY_PATH at a mounted secret, e.g.:
#   docker run -v "$PWD/private-key.pem:/etc/arietta/private-key.pem:ro" ...
ENV PORT=5050 \
    GITHUB_PRIVATE_KEY_PATH=/etc/arietta/private-key.pem

USER 10001:10001
EXPOSE 5050

ENTRYPOINT ["/usr/local/bin/arietta-bot"]
