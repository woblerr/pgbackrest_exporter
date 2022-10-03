#!/usr/bin/env bash

PORT="${1:-9854}"
EXPORTER_TLS="${2:-false}"
EXPORTER_AUTH="${3:-false}"

# Users for test basic auth.
AUTH_USER="test"
AUTH_PASSWORD="test"

# Use http or https.
case ${EXPORTER_TLS} in
    "false")
        EXPORTER_URL="http://localhost:${PORT}/metrics"
        CURL_FLAGS=""
        ;;
    "true")
        EXPORTER_URL="https://localhost:${PORT}/metrics"
        CURL_FLAGS="-k"
        ;;
    *)
        echo "[ERROR] incorect value: get=${EXPORTER_TLS}, want=true or false"
        exit 1
        ;;
esac

# Use basic auth or not.
case ${EXPORTER_AUTH} in
    "false")
        ;;
    "true")
        CURL_FLAGS+=" -u ${AUTH_USER}:${AUTH_PASSWORD}"
        ;;
    *)
        echo "[ERROR] incorect value: get=${EXPORTER_AUTH}, want=true or false"
        exit 1
        ;;
esac

# A simple test to check the number of metrics.
# Format: regex for metric | repetitions.
declare -a REGEX_LIST=(
    '^pgbackrest_backup_delta_bytes{.*}|3'
    '^pgbackrest_backup_duration_seconds{.*}|3'
    '^pgbackrest_backup_error_status{.*,backup_type="full",.*} 0$|2'
    '^pgbackrest_backup_error_status{.*,backup_type="diff",.*,repo_key="2".*} 1$|1'
    '^pgbackrest_backup_since_last_completion_seconds{.*}|3'
    '^pgbackrest_backup_databases{.*,backup_type="full",.*} 2|2'
    '^pgbackrest_backup_databases{.*,backup_type="diff",.*,repo_key="2".*} 2|1'
    '^pgbackrest_backup_last_databases{.*}|3'
    '^pgbackrest_backup_info{.*} 1$|3'
    '^pgbackrest_backup_repo_delta_bytes{.*}|3'
    '^pgbackrest_backup_repo_size_bytes{.*}|3'
    '^pgbackrest_backup_size_bytes{.*}|3'
    '^pgbackrest_exporter_info{.*} 1$|1'
    '^pgbackrest_repo_status{.*,repo_key="1".*} 0$|1'
    '^pgbackrest_repo_status{.*,repo_key="2".*} 0$|1'
    '^pgbackrest_stanza_status{.*} 0$|1'
    '^pgbackrest_wal_archive_status{.*,repo_key="1",.*}|1'
    '^pgbackrest_wal_archive_status{.*,repo_key="2",.*}|1'
)

# Check results.
for i in "${REGEX_LIST[@]}"
do
    regex=$(echo ${i} | cut -f1 -d'|')
    cnt=$(echo ${i} | cut -f2 -d'|')
    metric_cnt=$(curl -s ${CURL_FLAGS} ${EXPORTER_URL} | grep -E "${regex}" | wc -l | tr -d ' ')
    if [[ ${metric_cnt} != ${cnt} ]]; then
        echo "[ERROR] on regex '${regex}': get=${metric_cnt}, want=${cnt}"
        exit 1
    fi
done

echo "[INFO] all tests passed"
exit 0
