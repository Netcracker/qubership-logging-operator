# Build the pipeline test binary from the same source tree as the operator.
FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine3.24@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

COPY go.* /workspace/
COPY api/go.* api/

RUN go mod download && go -C api mod download

COPY api/ api/
COPY controllers/ controllers/
COPY test/fluent-pipeline/ test/fluent-pipeline/

RUN CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" GO111MODULE=on \
    go build -o fluent-pipeline-test ./test/fluent-pipeline

FROM alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a

ENV USER_UID=2001 \
    USER_NAME=fluent-pipeline-test \
    GROUP_NAME=fluent-pipeline-test

WORKDIR /

RUN addgroup "${GROUP_NAME}" && adduser -D -G "${GROUP_NAME}" -u "${USER_UID}" "${USER_NAME}"

COPY --from=builder --chown=${USER_UID}:${GROUP_NAME} /workspace/fluent-pipeline-test .

USER ${USER_UID}

ENTRYPOINT ["/fluent-pipeline-test"]
