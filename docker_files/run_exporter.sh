#!/usr/bin/env bash

set -e

# Basic command for execute pgbackrest_exporter.
EXPORTER_COMMAND="/etc/pgbackrest/pgbackrest_exporter \
--web.endpoint=${EXPORTER_ENDPOINT} \
--web.listen-address=127.0.0.1:${EXPORTER_PORT} \
--web.config.file=${EXPORTER_CONFIG} \
--collect.interval=${COLLECT_INTERVAL} \
--backrest.stanza-include=${STANZA_INCLUDE} \
--backrest.stanza-exclude=${STANZA_EXCLUDE} \
--backrest.backup-type=${BACKUP_TYPE}"

# Check variable for enabling additional labels for WAL metrics.
[ "${VERBOSE_WAL}" == "true" ] &&  EXPORTER_COMMAND="${EXPORTER_COMMAND} --backrest.verbose-wal"

# Check variable for exposing the number of databases in backups.
[ "${DATABASE_COUNT}" == "true" ] &&  EXPORTER_COMMAND="${EXPORTER_COMMAND} --backrest.database-count --backrest.database-parallel-processes=${DATABASE_PARALLEL_PROCESSES}"

# Check variable for exposing the number of databases in the latest backups.
[ "${DATABASE_COUNT_LATEST}" == "true" ] &&  EXPORTER_COMMAND="${EXPORTER_COMMAND} --backrest.database-count-latest"

# Execute the final command.
exec ${EXPORTER_COMMAND}
