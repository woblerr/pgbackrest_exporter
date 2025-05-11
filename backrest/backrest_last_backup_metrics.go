package backrest

import (
	"log/slog"
	"sync"
	"time"

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
			"block_incr",
			"stanza"})
	pgbrStanzaBackupLastDurationMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_duration_seconds",
		Help: "Backup duration for the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"block_incr",
			"stanza",
		})
	pgbrStanzaBackupLastDatabaseSizeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_size_bytes",
		Help: "Full uncompressed size of the database in the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"block_incr",
			"stanza"})
	pgbrStanzaBackupLastDatabaseBackupSizeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_delta_bytes",
		Help: "Amount of data in the database to actually backup in the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"block_incr",
			"stanza"})
	pgbrStanzaBackupLastRepoBackupSetSizeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_repo_size_bytes",
		Help: "Full compressed files size to restore the database from the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"block_incr",
			"stanza"})
	pgbrStanzaBackupLastRepoBackupSetSizeMapMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_repo_size_map_bytes",
		Help: "Size of block incremental map in the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"block_incr",
			"stanza"})
	pgbrStanzaBackupLastRepoBackupSizeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_repo_delta_bytes",
		Help: "Compressed files size in the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"block_incr",
			"stanza"})
	pgbrStanzaBackupLastRepoBackupSizeMapMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_repo_delta_map_bytes",
		Help: "Size of block incremental delta map in the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"block_incr",
			"stanza"})
	pgbrStanzaBackupLastErrorMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_error_status",
		Help: "Error status in the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"block_incr",
			"stanza"})
	pgbrStanzaBackupLastAnnotationsMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_annotations",
		Help: "Number of annotations in the last full, differential or incremental backup.",
	},
		[]string{
			"backup_type",
			"block_incr",
			"stanza",
		})
	// For json pgBackRest output
	pgbrStanzaBackupLastReferencesMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_references",
		Help: "Number of references to another backup (backup reference list) in the last full, differential or incremental backup.",
	},
		[]string{
			// Don't change this order.
			// See function processBackupReferencesCount().
			"ref_backup",
			"backup_type",
			"block_incr",
			"stanza"})
	pgbrStanzaBackupLastDatabasesMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_last_databases",
		Help: "Number of databases in the last full, differential or incremental backup.",
	},
		[]string{
			// Don't change this order.
			// See function processSpecificBackupData().
			"backup_type",
			"stanza",
			"block_incr",
		})
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
func getBackupLastMetrics(stanzaName string, lastBackups lastBackupsStruct, currentUnixTime int64, setUpMetricValueFun setUpMetricValueFunType, logger *slog.Logger) {
	for _, backup := range []backupStruct{lastBackups.full, lastBackups.diff, lastBackups.incr} {
		// Repo backup map size for last backups.
		setUpMetric(
			pgbrStanzaBackupLastRepoBackupSetSizeMapMetric,
			"pgbackrest_backup_last_repo_size_map_bytes",
			convertInt64PointerToFloat64(backup.backupRepoSizeMap),
			setUpMetricValueFun,
			logger,
			backup.backupType,
			backup.backupBlockIncr,
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
			backup.backupBlockIncr,
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
			backup.backupBlockIncr,
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
			backup.backupBlockIncr,
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
			backup.backupBlockIncr,
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
			backup.backupBlockIncr,
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
			backup.backupBlockIncr,
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
			backup.backupBlockIncr,
			stanzaName,
		)
		// Backup error status.
		setUpMetric(
			pgbrStanzaBackupLastErrorMetric,
			"pgbackrest_backup_last_error_status",
			convertBoolPointerToFloat64(backup.backupError),
			setUpMetricValueFun,
			logger,
			backup.backupType,
			backup.backupBlockIncr,
			stanzaName,
		)
		// Number of backup annotations.
		// Information about number of annotations in backup has appeared since pgBackRest v2.41.
		// If there are no annotations, the metric will be set to 0 for this backup.
		setUpMetric(
			pgbrStanzaBackupLastAnnotationsMetric,
			"pgbackrest_backup_last_annotations",
			convertAnnotationPointerToFloat64(backup.backupAnnotation),
			setUpMetricValueFun,
			logger,
			backup.backupType,
			backup.backupBlockIncr,
			stanzaName,
		)
		// Number of references to another backup (backup reference list).
		// For no-last backups, the metric is collected only if the flag is set.
		// For last backups, the metric is always collected.
		processBackupReferencesCount(
			backup.backupReference,
			"pgbackrest_backup_last_references",
			pgbrStanzaBackupLastReferencesMetric,
			setUpMetricValueFun,
			logger,
			backup.backupType,
			backup.backupBlockIncr,
			stanzaName)
	}
}

// Set backup metrics:
//   - pgbackrest_backup_last_databases
func getBackupLastDBCountMetrics(config, configIncludePath, stanzaName string, lastBackups lastBackupsStruct, setUpMetricValueFun setUpMetricValueFunType, logger *slog.Logger) {
	// For diff and incr run in parallel.
	var wg sync.WaitGroup
	// If name for diff backup is equal to full, there is no point in re-receiving data.
	if lastBackups.diff.backupLabel != lastBackups.full.backupLabel {
		wg.Add(1)
		go func(backupLabel, backupType, backupBlockIncr string) {
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
				logger,
				backupBlockIncr,
			)
		}(lastBackups.diff.backupLabel, lastBackups.diff.backupType, lastBackups.diff.backupBlockIncr)
	}
	// If name for diff backup is equal to full, there is no point in re-receiving data.
	if lastBackups.incr.backupLabel != lastBackups.diff.backupLabel {
		wg.Add(1)
		go func(backupLabel, backupType, backupBlockIncr string) {
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
				logger,
				backupBlockIncr,
			)
		}(lastBackups.incr.backupLabel, lastBackups.incr.backupType, lastBackups.incr.backupBlockIncr)
	}
	var metricValue float64 = 0
	// Try to get info for full backup.
	parseStanzaDataSpecific, err := getParsedSpecificBackupInfoData(config, configIncludePath, stanzaName, lastBackups.full.backupLabel, logger)
	if err != nil {
		logger.Error(
			"Get data from pgBackRest failed",
			"stanza", stanzaName,
			"backup", lastBackups.full.backupLabel,
			"err", err,
		)
	}
	if (len(parseStanzaDataSpecific) != 0 && len(parseStanzaDataSpecific[0].Backup) != 0) &&
		parseStanzaDataSpecific[0].Backup[0].DatabaseRef != nil {
		metricValue = convertDatabaseRefPointerToFloat(parseStanzaDataSpecific[0].Backup[0].DatabaseRef)
	} else {
		logger.Warn(
			"No backup data returned",
			"stanza", stanzaName,
			"backup", lastBackups.full.backupLabel,
		)
	}
	// Number of databases in the last full backup.
	setUpMetric(
		pgbrStanzaBackupLastDatabasesMetric,
		"pgbackrest_backup_last_databases",
		metricValue,
		setUpMetricValueFun,
		logger,
		lastBackups.full.backupType,
		stanzaName,
		lastBackups.full.backupBlockIncr,
	)
	if lastBackups.diff.backupLabel == lastBackups.full.backupLabel {
		setUpMetric(
			pgbrStanzaBackupLastDatabasesMetric,
			"pgbackrest_backup_last_databases",
			metricValue,
			setUpMetricValueFun,
			logger,
			lastBackups.diff.backupType,
			stanzaName,
			lastBackups.diff.backupBlockIncr,
		)
	}
	if lastBackups.incr.backupLabel == lastBackups.diff.backupLabel {
		setUpMetric(
			pgbrStanzaBackupLastDatabasesMetric,
			"pgbackrest_backup_last_databases",
			metricValue,
			setUpMetricValueFun,
			logger,
			lastBackups.incr.backupType,
			stanzaName,
			lastBackups.incr.backupBlockIncr,
		)
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
	pgbrStanzaBackupLastReferencesMetric.Reset()
}
