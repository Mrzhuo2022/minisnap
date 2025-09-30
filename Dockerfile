# syntax=docker/dockerfile:1
FROM golang:1.22.0-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ENV CGO_ENABLED=0 GOOS=linux
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -buildvcs=false -ldflags='-s -w' -o /out/minisnap ./cmd/server
# pre-create runtime directories since final image is distroless (no shell)
RUN mkdir -p /out/content

FROM scratch

WORKDIR /app
COPY --from=build --chown=65532:65532 /out/minisnap /app/minisnap
COPY templates /app/templates
COPY --from=build --chown=65532:65532 /out/content /app/content

ENV BIND_ADDR=":8080" \
    CONTENT_DIR="content"

EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/app/minisnap"]
