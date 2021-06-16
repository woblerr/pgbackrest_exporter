ARG BACKREST_VERSION="2.33"
FROM golang:1.16-buster AS builder
COPY . /build
WORKDIR /build
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -trimpath -o pgbackrest_exporter pgbackrest_exporter.go

FROM ghcr.io/woblerr/pgbackrest:${BACKREST_VERSION}
ENV EXPORTER_ENDPOINT="/metrics" \
    EXPORTER_PORT="9854" \
    COLLECT_INTERVAL=600
COPY --from=builder /build/pgbackrest_exporter /etc/pgbackrest/pgbackrest_exporter
RUN chmod 755 /etc/pgbackrest/pgbackrest_exporter
LABEL \
    org.opencontainers.image.version="${REPO_BUILD_TAG}" \
    org.opencontainers.image.source="https://github.com/woblerr/pgbackrest_exporter"
ENTRYPOINT ["/entrypoint.sh"]
CMD /etc/pgbackrest/pgbackrest_exporter \
        --prom.endpoint=${EXPORTER_ENDPOINT} \
        --prom.port=${EXPORTER_PORT} \
        --collect.interval=${COLLECT_INTERVAL}
EXPOSE ${EXPORTER_PORT}