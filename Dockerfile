FROM golang:alpine AS builder

RUN apk add --no-cache git
WORKDIR /build
COPY go.mod go.sum ./
RUN GOTOOLCHAIN=auto go mod download
COPY . .
RUN CGO_ENABLED=0 GOTOOLCHAIN=auto go build -ldflags="-s -w" -o cookit .

FROM alpine:latest

RUN apk add --no-cache \
    ca-certificates \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ttf-freefont \
    font-noto \
    dbus \
    && mkdir -p /app/data

COPY --from=builder /build/cookit /usr/local/bin/cookit
WORKDIR /app

ENV PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/usr/bin/chromium-browser
ENV DISPLAY=host.docker.internal:0

ENTRYPOINT ["cookit"]
