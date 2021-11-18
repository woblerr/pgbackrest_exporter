ARG BACKREST_VERSION="2.36"
ARG PG_VERSION="13"

FROM golang:1.17-buster AS builder
ARG REPO_BUILD_TAG
COPY . /build
WORKDIR /build
RUN CGO_ENABLED=0 go build \
        -mod=vendor -trimpath \
        -ldflags "-X main.version=${REPO_BUILD_TAG}" \
        -o pgbackrest_exporter pgbackrest_exporter.go

FROM ghcr.io/woblerr/pgbackrest:${BACKREST_VERSION}
ARG REPO_BUILD_TAG
ARG PG_VERSION
ENV BACKREST_USER="postgres" \
    BACKREST_GROUP="postgres" \
    EXPORTER_PORT="9854"
RUN apt-get update -y \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y \
        curl \
        ca-certificates \
        gnupg \
    && curl https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add - \
    && echo "deb http://apt.postgresql.org/pub/repos/apt/ focal-pgdg main" \
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
        /var/lib/pgbackrest/repo1 \
        /e2e_tests
COPY --chown=${BACKREST_USER}:${BACKREST_GROUP} \
        ./e2e_tests/postgresql.auto.conf \
        /var/lib/postgresql/${PG_VERSION}/main/postgresql.auto.conf
COPY ./e2e_tests/pgbackrest.conf /etc/pgbackrest/pgbackrest.conf
COPY --chmod=755 ./e2e_tests/prepare_e2e.sh /e2e_tests/prepare_e2e.sh
COPY --from=builder --chmod=755 /build/pgbackrest_exporter /etc/pgbackrest/pgbackrest_exporter
ENTRYPOINT ["/entrypoint.sh"]
CMD /e2e_tests/prepare_e2e.sh
EXPOSE ${EXPORTER_PORT}
