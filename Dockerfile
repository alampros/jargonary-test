# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM node:26-alpine AS frontend
WORKDIR /src/ui
COPY ui/package.json ui/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund
COPY ui/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=frontend /src/internal/webui/dist ./internal/webui/dist
RUN mkdir -p /out/data
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/jargonary ./cmd/jargonary

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/jargonary /jargonary
COPY --from=build --chown=65532:65532 /out/data /data
USER 65532:65532
WORKDIR /
EXPOSE 8080
ENTRYPOINT ["/jargonary"]
