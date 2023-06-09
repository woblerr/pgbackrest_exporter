package backrest

import (
	"time"

	"github.com/go-kit/log"
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
//   - pgbackrest_backup_last_databases
func getBackupLastMetrics(config, configIncludePath, stanzaName string, lastBackups lastBackupsStruct, backupDBCountLatest bool, currentUnixTime int64, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) {
	var (
		err                     error
		parseStanzaDataSpecific []stanza
	)
	lastBackups.full.backupType = "full"
	lastBackups.diff.backupType = "diff"
	lastBackups.incr.backupType = "incr"
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
		// If the calculation of the number of databases in latest backups is enabled.
		// Information about number of databases in specific backup has appeared since pgBackRest v2.41.
		// In versions < v2.41 this is missing and the metric does not need to be collected.
		// getParsedSpecificBackupInfoData will return error in this case.
		if backupDBCountLatest {
			// Try to get info for full backup.
			parseStanzaDataSpecific, err = getParsedSpecificBackupInfoData(config, configIncludePath, stanzaName, lastBackups.full.backupLabel, logger)
			if err == nil {
				// In a normal situation, only one element with one backup should be returned.
				// If more than one element or one backup is returned, there is may be a bug in pgBackRest.
				// If it's not a bug, then this part will need to be refactoring.
				// Use *[]struct() type for backup.DatabaseRef.
				if parseStanzaDataSpecific[0].Backup[0].DatabaseRef != nil {
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
				}
			}
			// If name for diff backup is equal to full, there is no point in re-receiving data.
			if lastBackups.diff.backupLabel != lastBackups.full.backupLabel {
				parseStanzaDataSpecific, err = getParsedSpecificBackupInfoData(config, configIncludePath, stanzaName, lastBackups.diff.backupLabel, logger)
			}
			if err == nil {
				if parseStanzaDataSpecific[0].Backup[0].DatabaseRef != nil {
					// Number of databases in the last full or differential backup.
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
			}
			// If name for incr backup is equal to diff, there is no point in re-receiving data.
			if lastBackups.incr.backupLabel != lastBackups.diff.backupLabel {
				parseStanzaDataSpecific, err = getParsedSpecificBackupInfoData(config, configIncludePath, stanzaName, lastBackups.incr.backupLabel, logger)
			}
			if err == nil {
				if parseStanzaDataSpecific[0].Backup[0].DatabaseRef != nil {
					// Number of databases in the last full, differential or incremental backup.
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
	}
}

func resetLastBackupMetrics() {
	pgbrStanzaBackupSinceLastCompletionSecondsMetric.Reset()
	pgbrStanzaBackupLastDatabasesMetric.Reset()
}
