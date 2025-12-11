#!/usr/bin/env bash

PORT="${1:-9854}"
EXPORTER_TLS="${2:-false}"
EXPORTER_AUTH="${3:-false}"
CERT_PATH="${4:-}"
MODE="${5:-}"

# Users for test basic auth.
AUTH_USER="test"
AUTH_PASSWORD="test"

# Cert auth.
AUTH_CERT="user.pem"
AUTH_KEY="user.key"

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

# Use basic auth, cert or not.
case ${EXPORTER_AUTH} in
    "false")
        ;;
    "basic")
        CURL_FLAGS+=" -u ${AUTH_USER}:${AUTH_PASSWORD}"
        ;;
    "cert")
        CURL_FLAGS+=" --cert ${CERT_PATH}/${AUTH_CERT} --key ${CERT_PATH}/${AUTH_KEY}"
        ;;
    *)
        echo "[ERROR] incorect value: get=${EXPORTER_AUTH}, want=false, basic or cert"
        exit 1
        ;;
esac

# A simple test to check the number of metrics.
# Format: regex for metric | repetitions.
#
# Additional comments:
#  - for '^pgbackrest_backup_last_annotations{.*} 0$|3' zero is correct,
#    because we create full and diff backups in repo 2 without any annotations.
case "${MODE}" in
    "exclude")
        declare -a REGEX_LIST=(
    '^pgbackrest_exporter_build_info{.*} 1$|1'
    '^pgbackrest_exporter_status{stanza="all-stanzas-except-excluded"} 1$|1'
        )
        ;;
    "include")
        declare -a REGEX_LIST=(
    '^pgbackrest_exporter_status{stanza="demo"} 1$|1'
    '^pgbackrest_stanza_status{stanza="demo"} 0$|1'
    '^pgbackrest_backup_last_size_bytes{backup_type="full",.*,stanza="demo"}|1'
        )
        ;;
    *)
        declare -a REGEX_LIST=(
    '^pgbackrest_backup_annotations{.*,backup_type="full",.*} 1$|1'
    '^pgbackrest_backup_databases{.*,backup_type="full",.*} 2$|2'
    '^pgbackrest_backup_databases{.*,backup_type="diff",.*,repo_key="2".*} 2$|1'
    '^pgbackrest_backup_delta_bytes{.*}|3'
    '^pgbackrest_backup_duration_seconds{.*}|3'
    '^pgbackrest_backup_error_status{.*,backup_type="full",.*} 0$|2'
    '^pgbackrest_backup_error_status{.*,backup_type="diff",.*,repo_key="2".*} 1$|1'
    '^pgbackrest_backup_info{.*,block_incr="n",.*} 1$|1'
    '^pgbackrest_backup_info{.*,block_incr="y",.*} 1$|2'
    '^pgbackrest_backup_last_annotations{.*} 0$|3'
    '^pgbackrest_backup_last_databases{.*}|3'
    '^pgbackrest_backup_last_delta_bytes{.*}|3'
    '^pgbackrest_backup_last_duration_seconds{.*}|3'
    '^pgbackrest_backup_last_error_status{backup_type="full",.*} 0$|1'
    '^pgbackrest_backup_last_error_status{backup_type="diff",.*} 1$|1'
    '^pgbackrest_backup_last_error_status{backup_type="incr",.*} 1$|1'
    '^pgbackrest_backup_last_references{backup_type="diff",.*,ref_backup="diff",.*} 0|1'
    '^pgbackrest_backup_last_references{backup_type="diff",.*,ref_backup="full",.*} 1|1'
    '^pgbackrest_backup_last_references{backup_type="diff",.*,ref_backup="incr",.*} 0|1'
    '^pgbackrest_backup_last_references{backup_type="full",.*} 0|3'
    '^pgbackrest_backup_last_references{backup_type="incr",.*,ref_backup="diff",.*} 0|1'
    '^pgbackrest_backup_last_references{backup_type="incr",.*,ref_backup="full",.*} 1|1'
    '^pgbackrest_backup_last_references{backup_type="incr",.*,ref_backup="incr",.*} 0|1'
    '^pgbackrest_backup_last_repo_delta_bytes{.*}|3'
    '^pgbackrest_backup_last_repo_delta_map_bytes{.*}|3'
    '^pgbackrest_backup_last_repo_size_bytes{.*} 0$|3'
    '^pgbackrest_backup_last_repo_size_map_bytes{.*}|3'
    '^pgbackrest_backup_last_size_bytes{.*}|3'
    '^pgbackrest_backup_references{.*,backup_type="diff",.*,ref_backup="full",.*} 1$|1'
    '^pgbackrest_backup_references{.*,backup_type="diff",.*,ref_backup="diff",.*} 0$|1'
    '^pgbackrest_backup_references{.*,backup_type="diff",.*,ref_backup="incr",.*} 0$|1'
    '^pgbackrest_backup_references{.*,backup_type="full",.*,repo_key="1",.*} 0$|3'
    '^pgbackrest_backup_references{.*,backup_type="full",.*,repo_key="2",.*} 0$|3'
    '^pgbackrest_backup_repo_delta_bytes{.*}|3'
    '^pgbackrest_backup_repo_delta_map_bytes{.*,repo_key="1",.*} 0$|1'
    '^pgbackrest_backup_repo_delta_map_bytes{.*,repo_key="2",.*}|2'
    '^pgbackrest_backup_repo_size_bytes{.*,repo_key="1",.*}|1'
    '^pgbackrest_backup_repo_size_bytes{.*,repo_key="2",.*} 0$|2'
    '^pgbackrest_backup_repo_size_map_bytes{.*,repo_key="1",.*} 0$|1'
    '^pgbackrest_backup_repo_size_map_bytes{.*,repo_key="2",.*}|2'
    '^pgbackrest_backup_since_last_completion_seconds{.*}|3'
    '^pgbackrest_backup_size_bytes{.*}|3'
    '^pgbackrest_exporter_build_info{.*} 1$|1'
    '^pgbackrest_exporter_status{stanza="all-stanzas"} 1$|1'
    '^pgbackrest_repo_status{.*,repo_key="1".*} 0$|1'
    '^pgbackrest_repo_status{.*,repo_key="2".*} 0$|1'
    '^pgbackrest_stanza_backup_compete_bytes{.*} 0$|1'
    '^pgbackrest_stanza_backup_total_bytes{.*} 0$|1'
    '^pgbackrest_stanza_lock_status{.*} 0$|1'
    '^pgbackrest_stanza_status{.*} 0$|1'
    '^pgbackrest_version_info|1'
    '^pgbackrest_wal_archive_status{.*,repo_key="1",.*}|1'
    '^pgbackrest_wal_archive_status{.*,repo_key="2",.*}|1'
        )
        ;;
esac

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
