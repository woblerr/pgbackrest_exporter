ARG BACKREST_VERSION="2.34"
ARG REPO_BUILD_TAG="unknown"

FROM golang:1.16-buster AS builder
ARG REPO_BUILD_TAG
COPY . /build
WORKDIR /build
RUN CGO_ENABLED=0 go build \
        -mod=vendor -trimpath \
        -ldflags "-X main.version=${REPO_BUILD_TAG}" \
        -o pgbackrest_exporter pgbackrest_exporter.go

FROM ghcr.io/woblerr/pgbackrest:${BACKREST_VERSION}
ARG REPO_BUILD_TAG
ENV EXPORTER_ENDPOINT="/metrics" \
    EXPORTER_PORT="9854" \
    STANZA="" \
    COLLECT_INTERVAL="600"
COPY --from=builder /build/pgbackrest_exporter /etc/pgbackrest/pgbackrest_exporter
RUN chmod 755 /etc/pgbackrest/pgbackrest_exporter
LABEL \
    org.opencontainers.image.version="${REPO_BUILD_TAG}" \
    org.opencontainers.image.source="https://github.com/woblerr/pgbackrest_exporter"
ENTRYPOINT ["/entrypoint.sh"]
CMD /etc/pgbackrest/pgbackrest_exporter \
        --prom.endpoint=${EXPORTER_ENDPOINT} \
        --prom.port=${EXPORTER_PORT} \
        --backrest.stanza-include=${STANZA} \
        --collect.interval=${COLLECT_INTERVAL}
EXPOSE ${EXPORTER_PORT}