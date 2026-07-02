# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.26.4-alpine3.22@sha256:727cfc3c40be55cd1bc9a4a059406b28a059857e3be752aa9d09531e12c20c56 AS builder

ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Copy the Go sources
COPY api/ api/
COPY controllers/ controllers/
COPY cmd/operator/main.go main.go
COPY go.* /workspace/

# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go work sync

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -a -o manager main.go

# Use alpine tiny images as a base
FROM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

ENV USER_UID=2001 \
    USER_NAME=logging-operator \
    GROUP_NAME=logging-operator

WORKDIR /
COPY --from=builder --chown=${USER_UID} /workspace/manager .

RUN addgroup ${GROUP_NAME} && adduser -D -G ${GROUP_NAME} -u ${USER_UID} ${USER_NAME}
USER ${USER_UID}

ENTRYPOINT ["/manager"]
