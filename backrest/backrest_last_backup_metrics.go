package backrest

import (
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Differential backup is always based on last full,
	// if the last backup was full, the metric will take full backup value.
	// Incremental backup is always based on last full or differential,
	// if the last backup was full or differential, the metric will take
	// full or differential backup value.
	pgbrStanzaBackupSinceLastCompletionSecondsMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_since_last_completion_seconds",
		Help: "Seconds since the last completed full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"stanza"})
	// Differential backup is always based on last full,
	// if the last backup was full, the metric will take full backup value.
	// Incremental backup is always based on last full or differential,
	// if the last backup was full or differential, the metric will take
	// full or differential backup value.
	pgbrStanzaBackupLastDatabasesMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_databases",
		Help: "Number of databases in the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"stanza"})
	pgbrStanzaBackupLastDurationMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_duration_seconds",
		Help: "Backup duration for the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"stanza",
		})
	pgbrStanzaBackupLastDatabaseSizeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_size_bytes",
		Help: "Full uncompressed size of the database in the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"stanza"})
	pgbrStanzaBackupLastDatabaseBackupSizeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_delta_bytes",
		Help: "Amount of data in the database to actually backup in the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"stanza"})
	pgbrStanzaBackupLastRepoBackupSetSizeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_repo_size_bytes",
		Help: "Full compressed files size to restore the database from the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"stanza"})
	pgbrStanzaBackupLastRepoBackupSetSizeMapMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_repo_size_map_bytes",
		Help: "Size of block incremental map in the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"stanza"})
	pgbrStanzaBackupLastRepoBackupSizeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_repo_delta_bytes",
		Help: "Compressed files size in the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"stanza"})
	pgbrStanzaBackupLastRepoBackupSizeMapMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_repo_delta_map_bytes",
		Help: "Size of block incremental delta map in the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"stanza"})
	pgbrStanzaBackupLastErrorMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_error_status",
		Help: "Error status in the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"stanza"})
	pgbrStanzaBackupLastAnnotationsMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_annotations",
		Help: "Number of annotations in the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"stanza"})
)

// Set backup metrics:
//   - pgbackrest_backup_since_last_completion_seconds
//   - pgbackrest_backup_last_duration_seconds
//   - pgbackrest_backup_last_size_bytes
//   - pgbackrest_backup_last_delta_bytes
//   - pgbackrest_backup_last_repo_size_bytes
//   - pgbackrest_backup_last_repo_size_map_bytes
//   - pgbackrest_backup_last_repo_delta_bytes
//   - pgbackrest_backup_last_repo_delta_map_bytes
//   - pgbackrest_backup_last_error_status
//   - pgbackrest_backup_last_annotations
func getBackupLastMetrics(stanzaName string, lastBackups lastBackupsStruct, currentUnixTime int64, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) {
	// If full backup exists, the values of metrics for differential and
	// incremental backups also will be set.
	// If not - metrics won't be set.
	if !lastBackups.full.backupTime.IsZero() {
		for _, backup := range []backupStruct{lastBackups.full, lastBackups.diff, lastBackups.incr} {
			// Repo backup map size for last backups.
			setUpMetric(
				pgbrStanzaBackupLastRepoBackupSetSizeMapMetric,
				"pgbackrest_backup_last_repo_size_map_bytes",
				convertInt64PointerToFloat64(backup.backupRepoSizeMap),
				setUpMetricValueFun,
				logger,
				backup.backupType,
				stanzaName,
			)
			// Repo backup delta map size for last backups.
			setUpMetric(
				pgbrStanzaBackupLastRepoBackupSizeMapMetric,
				"pgbackrest_backup_last_repo_delta_map_bytes",
				convertInt64PointerToFloat64(backup.backupRepoDeltaMap),
				setUpMetricValueFun,
				logger,
				backup.backupType,
				stanzaName,
			)
			// Seconds since the last completed backups.
			setUpMetric(
				pgbrStanzaBackupSinceLastCompletionSecondsMetric,
				"pgbackrest_backup_since_last_completion_seconds",
				time.Unix(currentUnixTime, 0).Sub(backup.backupTime).Seconds(),
				setUpMetricValueFun,
				logger,
				backup.backupType,
				stanzaName,
			)
			// Backup durations in seconds for last backups.
			setUpMetric(
				pgbrStanzaBackupLastDurationMetric,
				"pgbackrest_backup_last_duration_seconds",
				backup.backupDuration,
				setUpMetricValueFun,
				logger,
				backup.backupType,
				stanzaName,
			)
			// Database size for last backups.
			setUpMetric(
				pgbrStanzaBackupLastDatabaseSizeMetric,
				"pgbackrest_backup_last_size_bytes",
				float64(backup.backupSize),
				setUpMetricValueFun,
				logger,
				backup.backupType,
				stanzaName,
			)
			// Database backup size for last backups.
			setUpMetric(
				pgbrStanzaBackupLastDatabaseBackupSizeMetric,
				"pgbackrest_backup_last_delta_bytes",
				float64(backup.backupDelta),
				setUpMetricValueFun,
				logger,
				backup.backupType,
				stanzaName,
			)
			// Repo backup set size.
			setUpMetric(
				pgbrStanzaBackupLastRepoBackupSetSizeMetric,
				"pgbackrest_backup_last_repo_size_bytes",
				convertInt64PointerToFloat64(backup.backupRepoSize),
				setUpMetricValueFun,
				logger,
				backup.backupType,
				stanzaName,
			)
			// Repo backup size.
			setUpMetric(
				pgbrStanzaBackupLastRepoBackupSizeMetric,
				"pgbackrest_backup_last_repo_delta_bytes",
				float64(backup.backupRepoDelta),
				setUpMetricValueFun,
				logger,
				backup.backupType,
				stanzaName,
			)
			// Backup error status.
			if backup.backupError != nil {
				setUpMetric(
					pgbrStanzaBackupLastErrorMetric,
					"pgbackrest_backup_last_error_status",
					convertBoolToFloat64(*backup.backupError),
					setUpMetricValueFun,
					logger,
					backup.backupType,
					stanzaName,
				)
			}
			// Number of backup annotations.
			// Information about number of annotations in backup has appeared since pgBackRest v2.41.
			// The metric is always set.
			// For last backups, unlike specific backups, it makes sense to always specify
			// this metric so the metric is not lost, even if the backup does not have annotations.
			if backup.backupAnnotation != nil {
				setUpMetric(
					pgbrStanzaBackupLastAnnotationsMetric,
					"pgbackrest_backup_last_annotations",
					float64(len(*backup.backupAnnotation)),
					setUpMetricValueFun,
					logger,
					backup.backupType,
					stanzaName,
				)
			} else {
				setUpMetric(
					pgbrStanzaBackupLastAnnotationsMetric,
					"pgbackrest_backup_annotations",
					0,
					setUpMetricValueFun,
					logger,
					backup.backupType,
					stanzaName,
				)
			}
		}
	}
}

// Set backup metrics:
//   - pgbackrest_backup_last_databases
func getBackupLastDBCountMetrics(config, configIncludePath, stanzaName string, lastBackups lastBackupsStruct, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) {
	// For diff and incr run in parallel.
	var wg sync.WaitGroup
	// If name for diff backup is equal to full, there is no point in re-receiving data.
	if lastBackups.diff.backupLabel != lastBackups.full.backupLabel {
		wg.Add(1)
		go func(backupLabel, backupType string) {
			defer wg.Done()
			processSpecificBackupData(
				config,
				configIncludePath,
				stanzaName,
				backupLabel,
				backupType,
				"pgbackrest_backup_last_databases",
				pgbrStanzaBackupLastDatabasesMetric,
				setUpMetricValueFun,
				logger)
		}(lastBackups.diff.backupLabel, lastBackups.diff.backupType)
	}
	// If name for diff backup is equal to full, there is no point in re-receiving data.
	if lastBackups.incr.backupLabel != lastBackups.diff.backupLabel {
		wg.Add(1)
		go func(backupLabel, backupType string) {
			defer wg.Done()
			processSpecificBackupData(
				config,
				configIncludePath,
				stanzaName,
				backupLabel,
				backupType,
				"pgbackrest_backup_last_databases",
				pgbrStanzaBackupLastDatabasesMetric,
				setUpMetricValueFun,
				logger)
		}(lastBackups.incr.backupLabel, lastBackups.incr.backupType)
	}
	// Try to get info for full backup.
	parseStanzaDataSpecific, err := getParsedSpecificBackupInfoData(config, configIncludePath, stanzaName, lastBackups.full.backupLabel, logger)
	if err != nil {
		level.Error(logger).Log(
			"msg", "Get data from pgBackRest failed",
			"stanza", stanzaName,
			"backup", lastBackups.full.backupLabel,
			"err", err,
		)
	}
	if checkBackupDatabaseRef(parseStanzaDataSpecific) {
		// Number of databases in the last full backup.
		setUpMetric(
			pgbrStanzaBackupLastDatabasesMetric,
			"pgbackrest_backup_last_databases",
			float64(len(*parseStanzaDataSpecific[0].Backup[0].DatabaseRef)),
			setUpMetricValueFun,
			logger,
			lastBackups.full.backupType,
			stanzaName,
		)
		if lastBackups.diff.backupLabel == lastBackups.full.backupLabel {
			setUpMetric(
				pgbrStanzaBackupLastDatabasesMetric,
				"pgbackrest_backup_last_databases",
				float64(len(*parseStanzaDataSpecific[0].Backup[0].DatabaseRef)),
				setUpMetricValueFun,
				logger,
				lastBackups.diff.backupType,
				stanzaName,
			)
		}
		if lastBackups.incr.backupLabel == lastBackups.diff.backupLabel {
			setUpMetric(
				pgbrStanzaBackupLastDatabasesMetric,
				"pgbackrest_backup_last_databases",
				float64(len(*parseStanzaDataSpecific[0].Backup[0].DatabaseRef)),
				setUpMetricValueFun,
				logger,
				lastBackups.incr.backupType,
				stanzaName,
			)
		}
	}
}

func resetLastBackupMetrics() {
	pgbrStanzaBackupSinceLastCompletionSecondsMetric.Reset()
	pgbrStanzaBackupLastDatabasesMetric.Reset()
	pgbrStanzaBackupLastDurationMetric.Reset()
	pgbrStanzaBackupLastDatabaseSizeMetric.Reset()
	pgbrStanzaBackupLastDatabaseBackupSizeMetric.Reset()
	pgbrStanzaBackupLastRepoBackupSetSizeMetric.Reset()
	pgbrStanzaBackupLastRepoBackupSetSizeMapMetric.Reset()
	pgbrStanzaBackupLastRepoBackupSizeMetric.Reset()
	pgbrStanzaBackupLastRepoBackupSizeMapMetric.Reset()
	pgbrStanzaBackupLastErrorMetric.Reset()
	pgbrStanzaBackupLastAnnotationsMetric.Reset()
}
