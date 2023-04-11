# Build the manager binary
FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.17-alpine as builder

WORKDIR /src

COPY . /src

# Build
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -a -o /out/manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /out/manager .

USER 1001

ENTRYPOINT ["/manager"]
