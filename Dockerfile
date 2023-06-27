ARG BACKREST_VERSION="2.45"
ARG DOCKER_BACKREST_VERSION="v0.18"
ARG REPO_BUILD_TAG="unknown"

FROM golang:1.18-buster AS builder
ARG REPO_BUILD_TAG
COPY . /build
WORKDIR /build
RUN CGO_ENABLED=0 go build \
        -mod=vendor -trimpath \
        -ldflags "-X main.version=${REPO_BUILD_TAG}" \
        -o pgbackrest_exporter pgbackrest_exporter.go

FROM ghcr.io/woblerr/pgbackrest:${BACKREST_VERSION}-${DOCKER_BACKREST_VERSION}
ARG REPO_BUILD_TAG
ENV EXPORTER_ENDPOINT="/metrics" \
    EXPORTER_PORT="9854" \
    EXPORTER_CONFIG="" \
    STANZA_INCLUDE="" \
    STANZA_EXCLUDE="" \
    COLLECT_INTERVAL="600" \
    BACKUP_TYPE="" \
    VERBOSE_WAL="false" \
    DATABASE_COUNT="false" \
    DATABASE_PARALLEL_PROCESSES="1" \
    DATABASE_COUNT_LATEST="false"
COPY --chmod=755 docker_files/run_exporter.sh /run_exporter.sh
COPY --from=builder --chmod=755 /build/pgbackrest_exporter /etc/pgbackrest/pgbackrest_exporter
LABEL \
    org.opencontainers.image.version="${REPO_BUILD_TAG}" \
    org.opencontainers.image.source="https://github.com/woblerr/pgbackrest_exporter"
ENTRYPOINT ["/entrypoint.sh"]
CMD ["/run_exporter.sh"]
EXPOSE ${EXPORTER_PORT}