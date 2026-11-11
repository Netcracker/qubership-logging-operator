# Build the pipeline test binary from the same source tree as the operator.
FROM --platform=$BUILDPLATFORM golang:1.27.4-alpine3.24@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

COPY go.* /workspace/
COPY api/go.* api/

RUN go mod download && go -C api mod download

COPY api/ api/
COPY internal/ internal/
COPY test/fluent-pipeline/ test/fluent-pipeline/

RUN CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" GO111MODULE=on \
    go build -o fluent-pipeline-test ./test/fluent-pipeline

FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf

ENV USER_UID=2001 \
    USER_NAME=fluent-pipeline-test \
    GROUP_NAME=fluent-pipeline-test

WORKDIR /

RUN addgroup "${GROUP_NAME}" && adduser -D -G "${GROUP_NAME}" -u "${USER_UID}" "${USER_NAME}"

COPY --from=builder --chown=${USER_UID}:${GROUP_NAME} /workspace/fluent-pipeline-test .

USER ${USER_UID}

ENTRYPOINT ["/fluent-pipeline-test"]
