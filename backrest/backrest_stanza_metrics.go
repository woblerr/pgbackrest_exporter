package backrest

import (
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	pgbrStanzaStatusMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_stanza_status",
		Help: "Current stanza status.",
	},
		[]string{"stanza"})
	pgbrStanzaLockStatusMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_stanza_lock_status",
		Help: "Current stanza lock status.",
	},
		[]string{"stanza"})
	pgbrStanzaBackupInProgressCompleteMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_stanza_backup_compete_bytes",
		Help: "Completed size for backup in progress.",
	},
		[]string{"stanza"})
	pgbrStanzaBackupInProgressTotalMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_stanza_backup_total_bytes",
		Help: "Total size for backup in progress.",
	},
		[]string{"stanza"})
)

// Set stanza metrics:
//   - pgbackrest_stanza_status
func getStanzaMetrics(stanzaName string, stanzaStatus status, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) {
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
	// Stanza lock statuses.
	// It's  not currently possible to know which command is running: backup, expire, or stanza-*.
	// The stanza commands are very unlikely to be running so it's pretty safe to guess backup/expire.
	// If there is information about lock, then with a very high probability it is backup/expire.
	// See https://github.com/pgbackrest/pgbackrest/commit/e92eb709d6c60e44f406af139b8346b12fee1868
	// If the values stanzaStatus.Lock.Backup.SizeTotal and stanzaStatus.Lock.Backup.SizeComplete (start from pgBackrest v2.48) are specified,
	// then it's exactly backup. However, pgBackRest currently don't return any statuses in this place.
	// May be this functionality will be added in the future.
	// When creating dashboards, this should be remembered.
	setUpMetric(
		pgbrStanzaStatusMetric,
		"pgbackrest_stanza_lock_status",
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
		"pgbackrest_stanza_backup_compete_bytes",
		convertInt64PointerToFloat64(stanzaStatus.Lock.Backup.SizeComplete),
		setUpMetricValueFun,
		logger,
		stanzaName,
	)
}

func resetStanzaMetrics() {
	pgbrStanzaStatusMetric.Reset()
}
