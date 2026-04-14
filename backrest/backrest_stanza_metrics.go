package backrest

import (
	"log/slog"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	pgbrStanzaStatusMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_stanza_status",
		Help: "Current stanza status.",
	},
		[]string{"stanza"})
	pgbrStanzaBackupLockStatusMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_stanza_backup_lock_status",
		Help: "Current stanza backup lock status.",
	},
		[]string{"stanza"})
	pgbrStanzaBackupInProgressCompleteMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_stanza_backup_complete_bytes",
		Help: "Completed size for backup in progress.",
	},
		[]string{"stanza"})
	pgbrStanzaBackupInProgressTotalMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_stanza_backup_total_bytes",
		Help: "Total size for backup in progress.",
	},
		[]string{"stanza"})
	pgbrStanzaRestoreLockStatusMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_stanza_restore_lock_status",
		Help: "Current stanza restore lock status.",
	},
		[]string{"stanza"})
	pgbrStanzaRestoreInProgressCompleteMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_stanza_restore_complete_bytes",
		Help: "Completed size for restore in progress.",
	},
		[]string{"stanza"})
	pgbrStanzaRestoreInProgressTotalMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_stanza_restore_total_bytes",
		Help: "Total size for restore in progress.",
	},
		[]string{"stanza"})
	pgbrStanzaBackupInProgressRepoCompleteMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_stanza_backup_repo_complete_bytes",
		Help: "Completed size for backup in progress per repository.",
	},
		[]string{"repo_key", "stanza"})
	pgbrStanzaBackupInProgressRepoTotalMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_stanza_backup_repo_total_bytes",
		Help: "Total size for backup in progress per repository.",
	},
		[]string{"repo_key", "stanza"})
)

// Set stanza metrics:
//   - pgbackrest_stanza_status
func getStanzaMetrics(stanzaName string, stanzaStatus status, setUpMetricValueFun setUpMetricValueFunType, logger *slog.Logger) {
	//https://github.com/pgbackrest/pgbackrest/blob/03021c6a17f1374e84ef42614fa1dd2a6be4b64d/src/command/info/info.c#L78-L94
	// Stanza statuses:
	//  0: "ok",
	//  1: "missing stanza path",
	//  2: "no valid backups",
	//  3: "missing stanza data",
	//  4: "different across repos",
	//  5: "database mismatch across repos",
	//  6: "requested backup not found",
	//  99: "other".
	setUpMetric(
		pgbrStanzaStatusMetric,
		"pgbackrest_stanza_status",
		float64(stanzaStatus.Code),
		setUpMetricValueFun,
		logger,
		stanzaName,
	)
	// Stanza lock backup statuses.
	// It's  not currently possible to know which command is running: backup, expire, or stanza-*.
	// The stanza commands are very unlikely to be running so it's pretty safe to guess backup/expire.
	// If there is information about lock, then with a very high probability it is backup/expire.
	// See https://github.com/pgbackrest/pgbackrest/commit/e92eb709d6c60e44f406af139b8346b12fee1868
	// If the values stanzaStatus.Lock.Backup.SizeTotal and stanzaStatus.Lock.Backup.SizeComplete (start from pgBackrest v2.48) are specified,
	// then it's exactly backup. However, pgBackRest currently don't return any statuses in this place.
	// May be this functionality will be added in the future.
	// When creating dashboards, this should be remembered.
	// Stanza lock backup statuses:
	//  0: "no active operation with stanza",
	//  1: one of the commands is running for stanza: backup, expire or stanza-*".
	setUpMetric(
		pgbrStanzaBackupLockStatusMetric,
		"pgbackrest_stanza_backup_lock_status",
		convertBoolToFloat64(stanzaStatus.Lock.Backup.Held),
		setUpMetricValueFun,
		logger,
		stanzaName,
	)
	// For pgBackRest >= v2.48 these metrics can have relevant values.
	// For pgBackRest < v2.48 - they will always have the value 0.
	// When backup in progress information is displayed in them.
	// It is convenient in monitoring to display the percentage of completion of the backup process.
	setUpMetric(
		pgbrStanzaBackupInProgressTotalMetric,
		"pgbackrest_stanza_backup_total_bytes",
		convertInt64PointerToFloat64(stanzaStatus.Lock.Backup.SizeTotal),
		setUpMetricValueFun,
		logger,
		stanzaName,
	)
	setUpMetric(
		pgbrStanzaBackupInProgressCompleteMetric,
		"pgbackrest_stanza_backup_complete_bytes",
		convertInt64PointerToFloat64(stanzaStatus.Lock.Backup.SizeComplete),
		setUpMetricValueFun,
		logger,
		stanzaName,
	)
	// For pgBackRest >= v2.56.0 these metrics can have relevant values.
	// For pgBackRest < v2.56.0 - they will always have the value 0.
	// See https://github.com/pgbackrest/pgbackrest/commit/8cdd9ce1c4ab6cca508932a41a3013374d7547ef
	// When restore in progress information is displayed in them.
	// It is convenient in monitoring to display the percentage of completion of the restore process.
	// Stanza lock restore statuses:
	//  0: "no active restore",
	//  1: restore in progress".
	setUpMetric(
		pgbrStanzaRestoreLockStatusMetric,
		"pgbackrest_stanza_restore_lock_status",
		convertBoolToFloat64(stanzaStatus.Lock.Restore.Held),
		setUpMetricValueFun,
		logger,
		stanzaName,
	)
	setUpMetric(
		pgbrStanzaRestoreInProgressTotalMetric,
		"pgbackrest_stanza_restore_total_bytes",
		convertInt64PointerToFloat64(stanzaStatus.Lock.Restore.SizeTotal),
		setUpMetricValueFun,
		logger,
		stanzaName,
	)
	setUpMetric(
		pgbrStanzaRestoreInProgressCompleteMetric,
		"pgbackrest_stanza_restore_complete_bytes",
		convertInt64PointerToFloat64(stanzaStatus.Lock.Restore.SizeComplete),
		setUpMetricValueFun,
		logger,
		stanzaName,
	)
	// For pgBackRest >= v2.59 these metrics can have relevant values per repo.
	// For pgBackRest < v2.59 - they will always have the value 0 with repo_key="0".
	for _, repoLock := range convertBackupLockRepoPointerToSlice(stanzaStatus.Lock.Backup.Repo) {
		repoKey := strconv.Itoa(repoLock.Key)
		setUpMetric(
			pgbrStanzaBackupInProgressRepoTotalMetric,
			"pgbackrest_stanza_backup_repo_total_bytes",
			float64(repoLock.SizeTotal),
			setUpMetricValueFun,
			logger,
			repoKey,
			stanzaName,
		)
		setUpMetric(
			pgbrStanzaBackupInProgressRepoCompleteMetric,
			"pgbackrest_stanza_backup_repo_complete_bytes",
			float64(repoLock.SizeComplete),
			setUpMetricValueFun,
			logger,
			repoKey,
			stanzaName,
		)
	}
}

func resetStanzaMetrics() {
	pgbrStanzaStatusMetric.Reset()
	pgbrStanzaBackupLockStatusMetric.Reset()
	pgbrStanzaBackupInProgressTotalMetric.Reset()
	pgbrStanzaBackupInProgressCompleteMetric.Reset()
	pgbrStanzaBackupInProgressRepoTotalMetric.Reset()
	pgbrStanzaBackupInProgressRepoCompleteMetric.Reset()
	pgbrStanzaRestoreLockStatusMetric.Reset()
	pgbrStanzaRestoreInProgressTotalMetric.Reset()
	pgbrStanzaRestoreInProgressCompleteMetric.Reset()
}
