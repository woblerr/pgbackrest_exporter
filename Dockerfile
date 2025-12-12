ARG BACKREST_VERSION="2.56.0"
ARG DOCKER_BACKREST_VERSION="v0.34"
ARG REPO_BUILD_TAG="unknown"

FROM golang:1.24-bookworm AS builder
ARG REPO_BUILD_TAG
COPY . /build
WORKDIR /build
RUN CGO_ENABLED=0 go build \
        -mod=vendor -trimpath \
        -ldflags "-s -w \
            -X github.com/prometheus/common/version.Version=${REPO_BUILD_TAG} \
            -X github.com/prometheus/common/version.BuildDate=$(date +%Y-%m-%dT%H:%M:%S%z) \
            -X github.com/prometheus/common/version.Branch=$(git rev-parse --abbrev-ref HEAD) \
            -X github.com/prometheus/common/version.Revision=$(git rev-parse --short HEAD) \
            -X github.com/prometheus/common/version.BuildUser=pgbackrest_exporter" \
        -o pgbackrest_exporter pgbackrest_exporter.go

FROM ghcr.io/woblerr/pgbackrest:${BACKREST_VERSION}-${DOCKER_BACKREST_VERSION}
ARG REPO_BUILD_TAG
ENV EXPORTER_TELEMETRY_PATH="/metrics" \
    EXPORTER_PORT="9854" \
    EXPORTER_CONFIG="" \
    STANZA_INCLUDE="" \
    STANZA_EXCLUDE="" \
    COLLECT_INTERVAL="600" \
    BACKUP_TYPE="" \
    VERBOSE_WAL="false" \
    DATABASE_COUNT="false" \
    DATABASE_PARALLEL_PROCESSES="1" \
    DATABASE_COUNT_LATEST="false" \
    COLLECTOR_PGBACKREST="true"
COPY --chmod=755 docker_files/run_exporter.sh /run_exporter.sh
COPY --from=builder --chmod=755 /build/pgbackrest_exporter /etc/pgbackrest/pgbackrest_exporter
LABEL \
    org.opencontainers.image.version="${REPO_BUILD_TAG}" \
    org.opencontainers.image.source="https://github.com/woblerr/pgbackrest_exporter"
ENTRYPOINT ["/entrypoint.sh"]
CMD ["/run_exporter.sh"]
EXPOSE ${EXPORTER_PORT}