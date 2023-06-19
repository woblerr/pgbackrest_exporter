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
)

// Set backup metrics:
//   - pgbackrest_backup_since_last_completion_seconds
func getBackupLastMetrics(stanzaName string, lastBackups lastBackupsStruct, currentUnixTime int64, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) {
	// If full backup exists, the values of metrics for differential and
	// incremental backups also will be set.
	// If not - metrics won't be set.
	if !lastBackups.full.backupTime.IsZero() {
		// Seconds since the last completed full backup.
		setUpMetric(
			pgbrStanzaBackupSinceLastCompletionSecondsMetric,
			"pgbackrest_backup_since_last_completion_seconds",
			time.Unix(currentUnixTime, 0).Sub(lastBackups.full.backupTime).Seconds(),
			setUpMetricValueFun,
			logger,
			lastBackups.full.backupType,
			stanzaName,
		)
		// Seconds since the last completed full or differential backup.
		setUpMetric(
			pgbrStanzaBackupSinceLastCompletionSecondsMetric,
			"pgbackrest_backup_since_last_completion_seconds",
			time.Unix(currentUnixTime, 0).Sub(lastBackups.diff.backupTime).Seconds(),
			setUpMetricValueFun,
			logger,
			lastBackups.diff.backupType,
			stanzaName,
		)
		// Seconds since the last completed full, differential or incremental backup.
		setUpMetric(
			pgbrStanzaBackupSinceLastCompletionSecondsMetric,
			"pgbackrest_backup_since_last_completion_seconds",
			time.Unix(currentUnixTime, 0).Sub(lastBackups.incr.backupTime).Seconds(),
			setUpMetricValueFun,
			logger,
			lastBackups.incr.backupType,
			stanzaName,
		)
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
		go processSpecificBackupData(config, configIncludePath, stanzaName, lastBackups.diff.backupLabel, lastBackups.diff.backupType, setUpMetricValueFun, logger, &wg)
	}
	// If name for diff backup is equal to full, there is no point in re-receiving data.
	if lastBackups.incr.backupLabel != lastBackups.diff.backupLabel {
		wg.Add(1)
		go processSpecificBackupData(config, configIncludePath, stanzaName, lastBackups.incr.backupLabel, lastBackups.incr.backupType, setUpMetricValueFun, logger, &wg)
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

func processSpecificBackupData(config, configIncludePath, stanzaName, backupLabel, backupType string, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger, wg *sync.WaitGroup) {
	defer wg.Done()
	parseStanzaDataSpecific, err := getParsedSpecificBackupInfoData(config, configIncludePath, stanzaName, backupLabel, logger)
	if err != nil {
		level.Error(logger).Log(
			"msg", "Get data from pgBackRest failed",
			"stanza", stanzaName,
			"backup", backupLabel,
			"err", err,
		)
	}
	if err == nil {
		if checkBackupDatabaseRef(parseStanzaDataSpecific) {
			// Number of databases in the last differential or incremental backup.
			setUpMetric(
				pgbrStanzaBackupLastDatabasesMetric,
				"pgbackrest_backup_last_databases",
				float64(len(*parseStanzaDataSpecific[0].Backup[0].DatabaseRef)),
				setUpMetricValueFun,
				logger,
				backupType,
				stanzaName,
			)
		}
	}
}

func resetLastBackupMetrics() {
	pgbrStanzaBackupSinceLastCompletionSecondsMetric.Reset()
	pgbrStanzaBackupLastDatabasesMetric.Reset()
}
