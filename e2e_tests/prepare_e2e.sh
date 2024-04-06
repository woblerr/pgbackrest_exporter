#!/usr/bin/env bash

# Exit on errors and on command pipe failures.
set -e

EXPORTER_CONFIG="${1}"

PG_CLUSTER="main"
PG_DATABASE="test_db"
PG_BIN="/usr/lib/postgresql/16/bin"
PG_DATA="/var/lib/postgresql/16/${PG_CLUSTER}"
BACKREST_STANZA="demo"
EXPORTER_COMMAND="/etc/pgbackrest/pgbackrest_exporter \
--backrest.database-count \
--backrest.database-parallel-processes=2 \
--backrest.database-count-latest"

# Enable checksums.
${PG_BIN}/pg_checksums -e -D ${PG_DATA}
# Start postgres.
pg_ctlcluster 16 ${PG_CLUSTER} start
# Create  database.
psql -c "create database ${PG_DATABASE}"
db_oid=$(psql -t -c "select OID from pg_database where datname='demo_db';")
# Create stanza.
pgbackrest stanza-create --stanza ${BACKREST_STANZA} --log-level-console warn
# Create full backup for stanza in repo1.
pgbackrest backup --stanza ${BACKREST_STANZA} --type full --repo 1 --log-level-console warn
# Get the name of created backup in repo1.
backup_name=$(pgbackrest info --output json --type full --repo 1 --log-level-console warn | jq --raw-output '.[].backup[].label')
# Create annotation for the backup in repo1.
pgbackrest annotate --stanza ${BACKREST_STANZA} --annotation "testkey=testvalue" --set ${backup_name} --repo 1 --log-level-console warn
# Create full bakup for stanza in repo2 with block incremental.
pgbackrest backup --stanza ${BACKREST_STANZA} --type full --repo 2 --repo2-bundle --repo2-block --log-level-console warn
# Currupt database file.
db_file=$(find ${PG_DATA}/base/${db_oid} -type f -regextype egrep -regex '.*/([0-9]){4}$' -print | head -n 1)
echo "currupt" >> ${db_file} 
# Create diff backup with corrupted databse file in repo2 with block incremental.
pgbackrest backup --stanza ${BACKREST_STANZA} --type diff --repo 2 --repo2-bundle --repo2-block --log-level-console warn
# Update exporter params.
[[ ! -z ${EXPORTER_CONFIG} ]] && EXPORTER_COMMAND="${EXPORTER_COMMAND} --web.config.file=${EXPORTER_CONFIG}"
# Run pgbackrest_exporter.
exec ${EXPORTER_COMMAND}