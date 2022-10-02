#!/usr/bin/env bash

set -e

# Basic command for execute pgbackrest_exporter.
EXPORTER_COMMAND="/etc/pgbackrest/pgbackrest_exporter \
--prom.endpoint=${EXPORTER_ENDPOINT} \
--prom.port=${EXPORTER_PORT} \
--collect.interval=${COLLECT_INTERVAL} \
--backrest.stanza-include=${STANZA_INCLUDE} \
--backrest.stanza-exclude=${STANZA_EXCLUDE} \
--backrest.backup-type=${BACKUP_TYPE}"

# Check variable for enabling additional labels for WAL metrics.
[ "${VERBOSE_WAL}" == "true" ] &&  EXPORTER_COMMAND="${EXPORTER_COMMAND} --backrest.verbose-wal"

# Check variable for exposing the number of databases in the latest backups.
[ "${DATABASE_COUNT_LATEST}" == "true" ] &&  EXPORTER_COMMAND="${EXPORTER_COMMAND} --backrest.database-count-latest"

# Execute the final command.
$(${EXPORTER_COMMAND})
