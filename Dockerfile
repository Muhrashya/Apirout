# Stage 1: Build React Vite Admin Console
FROM node:20-alpine AS web-builder

WORKDIR /web
COPY web/package*.json ./
RUN npm ci

ARG VERSION=1.0.0
ENV VITE_APP_VERSION=${VERSION}
COPY web/ ./
RUN npm run build

# Stage 2: Build Go Apirout binary
FROM golang:alpine AS go-builder

ARG VERSION=1.0.0
ENV GOTOOLCHAIN=auto
WORKDIR /build
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum* ./
RUN go mod download || true

COPY . /build
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o apirout ./cmd/apirout

# Stage 3: Minimal Production Alpine runtime
FROM alpine:3.21

WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata curl && \
    addgroup -S -g 10001 apirout && \
    adduser -S -u 10001 -G apirout -h /app apirout && \
    mkdir -p /app/data && \
    chown -R apirout:apirout /app

COPY --from=go-builder --chown=apirout:apirout /build/apirout /app/apirout
COPY --from=web-builder --chown=apirout:apirout /web/dist /app/web/dist

USER apirout

ENV STATIC_DIR=/app/web/dist
ENV DATA_DIR=/app/data
EXPOSE 8000 8001 8002 8100

ENTRYPOINT ["/app/apirout"]

