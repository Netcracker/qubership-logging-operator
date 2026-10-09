# Build the pipeline test binary from the same source tree as the operator.
FROM --platform=$BUILDPLATFORM golang:1.26.6-alpine3.24@sha256:af8d6740070b8906d12eae1c3e3ea0957fb63f492051ea05e354c38ef9fe88df AS builder

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

FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

ENV USER_UID=2001 \
    USER_NAME=fluent-pipeline-test \
    GROUP_NAME=fluent-pipeline-test

WORKDIR /

RUN addgroup "${GROUP_NAME}" && adduser -D -G "${GROUP_NAME}" -u "${USER_UID}" "${USER_NAME}"

COPY --from=builder --chown=${USER_UID}:${GROUP_NAME} /workspace/fluent-pipeline-test .

USER ${USER_UID}

ENTRYPOINT ["/fluent-pipeline-test"]
