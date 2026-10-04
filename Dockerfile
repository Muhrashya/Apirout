# Stage 1: Build React Vite Admin Console
FROM node:20-alpine AS web-builder

WORKDIR /web
COPY web/package*.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# Stage 2: Build Go Apirout binary
FROM golang:alpine AS go-builder

ENV GOTOOLCHAIN=auto
WORKDIR /build
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum* ./
RUN go mod download || true

COPY . /build
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o apirout ./cmd/apirout

# Stage 3: Minimal Production Alpine runtime
FROM alpine:3.19

WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata curl

COPY --from=go-builder /build/apirout /app/apirout
COPY --from=web-builder /web/dist /app/web/dist

ENV STATIC_DIR=/app/web/dist
EXPOSE 8000 8001 8002 8100

ENTRYPOINT ["/app/apirout"]
