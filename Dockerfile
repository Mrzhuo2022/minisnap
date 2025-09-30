# syntax=docker/dockerfile:1
FROM golang:1.22.0-alpine3.20 AS build

WORKDIR /src
COPY go.mod ./
COPY go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /out/minisnap ./cmd/server

FROM alpine:3.21

WORKDIR /app
COPY --from=build /out/minisnap ./minisnap
COPY templates ./templates
# 在首次运行时如果宿主机未挂载，会自动生成 content 目录
RUN mkdir -p /app/content

ENV BIND_ADDR=":8080" \
    CONTENT_DIR="content"

EXPOSE 8080

ENTRYPOINT ["/app/minisnap"]
