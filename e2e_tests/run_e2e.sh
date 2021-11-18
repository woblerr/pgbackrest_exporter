#!/usr/bin/env bash

PORT="${1}"
[[ -z ${PORT} ]] && exit 1

EXPORTER_URL="http://localhost:${PORT}/metrics"
# A simple test to check the number of metrics.
# Format: regex for metric | repetitions.
declare -a REGEX_LIST=(
    '^pgbackrest_backup_delta_bytes{.*}|3'
    '^pgbackrest_backup_duration_seconds{.*}|3'
    '^pgbackrest_backup_error_status{.*,backup_type="full",.*} 0$|2'
    '^pgbackrest_backup_error_status{.*,backup_type="diff",.*,repo_key="2".*} 1$|1'
    '^pgbackrest_backup_diff_since_last_completion_seconds{.*}|1'
    '^pgbackrest_backup_full_since_last_completion_seconds{.*}|1'
    '^pgbackrest_backup_incr_since_last_completion_seconds{.*}|1'
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
    metric_cnt=$(curl -s ${EXPORTER_URL} | grep -E "${regex}" | wc -l | tr -d ' ')
    if [[ ${metric_cnt} != ${cnt} ]]; then
        echo "[ERROR] on regex '${regex}': get=${metric_cnt}, want=${cnt}"
        exit 1
    fi
done

echo "[INFO] all tests passed"
exit 0
