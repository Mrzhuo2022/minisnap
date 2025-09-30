# syntax=docker/dockerfile:1
FROM golang:1.22.0-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ENV CGO_ENABLED=0 GOOS=linux
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags='-s -w' -o /out/minisnap ./cmd/server
# pre-create runtime directories since final image is distroless (no shell)
RUN mkdir -p /out/content

FROM gcr.io/distroless/static:nonroot

WORKDIR /app
COPY --from=build /out/minisnap /app/minisnap
COPY templates /app/templates
COPY --from=build /out/content /app/content

ENV BIND_ADDR=":8080" \
    CONTENT_DIR="content"

EXPOSE 8080

ENTRYPOINT ["/app/minisnap"]
