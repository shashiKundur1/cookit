FROM golang:latest AS builder

ENV GOTOOLCHAIN=auto
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o cookit .

FROM mcr.microsoft.com/playwright:v1.52.0-noble

USER root

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /build/cookit /usr/local/bin/cookit

RUN mkdir -p /app/data
WORKDIR /app

ENV DISPLAY=host.docker.internal:0

ENTRYPOINT ["cookit"]
