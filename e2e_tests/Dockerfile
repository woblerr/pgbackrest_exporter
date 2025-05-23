ARG BACKREST_VERSION="2.55.1"
ARG DOCKER_BACKREST_VERSION="v0.33"
ARG PG_VERSION="16"

FROM golang:1.23-bookworm AS builder
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
ARG PG_VERSION
ENV BACKREST_USER="postgres" \
    BACKREST_GROUP="postgres" \
    EXPORTER_PORT="9854" \
    EXPORTER_CONFIG=""
RUN apt-get update -y \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y \
        curl \
        ca-certificates \
        gnupg \
        jq \
        lsb-release \
    && curl https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /etc/apt/trusted.gpg.d/postgresql.gpg \
    && echo "deb http://apt.postgresql.org/pub/repos/apt/ $(lsb_release -cs)-pgdg main" \
        > /etc/apt/sources.list.d/pgdg.list
RUN apt-get update -y \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y \
        postgresql-${PG_VERSION} \
        postgresql-contrib-${PG_VERSION} \
    && apt-get autoremove -y \
    && apt-get autopurge -y \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p \
        /var/lib/pgbackrest/repo1 \
        /var/lib/pgbackrest/repo2 \
        /e2e_tests
COPY --chown=${BACKREST_USER}:${BACKREST_GROUP} \
        ./e2e_tests/postgresql.auto.conf \
        /var/lib/postgresql/${PG_VERSION}/main/postgresql.auto.conf
COPY ./e2e_tests/pgbackrest.conf /etc/pgbackrest/pgbackrest.conf
COPY --chown=${BACKREST_USER}:${BACKREST_GROUP} --chmod=400 ./e2e_tests/server.* /e2e_tests/
COPY --chown=${BACKREST_USER}:${BACKREST_GROUP} --chmod=400 ./e2e_tests/user.* /e2e_tests/
COPY --chown=${BACKREST_USER}:${BACKREST_GROUP} --chmod=644 ./e2e_tests/web_config_*.yml /e2e_tests/
COPY --chown=${BACKREST_USER}:${BACKREST_GROUP} --chmod=755 ./e2e_tests/prepare_e2e.sh /e2e_tests/prepare_e2e.sh
COPY --from=builder --chmod=755 /build/pgbackrest_exporter /etc/pgbackrest/pgbackrest_exporter
ENTRYPOINT ["/entrypoint.sh"]
CMD /e2e_tests/prepare_e2e.sh ${EXPORTER_CONFIG}
EXPOSE ${EXPORTER_PORT}
