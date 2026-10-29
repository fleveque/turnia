# syntax=docker/dockerfile:1

# The whole backend as one static binary on a distroless base (ADR 0004).
# The frontend joins it in milestone 14. `make image` builds it locally;
# Kamal builds it for the server (arm64) from milestone 4.3.

# The build stage runs on the building machine's own platform and Go
# cross-compiles for the one asked for, so an arm64 image builds on an amd64
# laptop without emulating arm64.
FROM --platform=$BUILDPLATFORM golang:1.27.1 AS build
ARG TARGETOS TARGETARCH
# The commit being built, which `turnia version` prints. Images are built
# from a checkout without .git, so Go can't find it by itself.
ARG REVISION=""

WORKDIR /src/backend
COPY backend/ ./
# The cache mounts keep downloaded modules and compiled packages between
# builds, without putting them in the image.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w -X main.revision=${REVISION}" -o /out/turnia ./cmd/turnia

# No shell, no package manager, no libc: the binary, CA certificates and
# time zone data, run as an unprivileged user.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/turnia /turnia
EXPOSE 8080
ENTRYPOINT ["/turnia"]
CMD ["serve"]
