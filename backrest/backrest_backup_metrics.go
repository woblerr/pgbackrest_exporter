package backrest

import (
	"strconv"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	pgbrStanzaBackupInfoMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_info",
		Help: "Backup info.",
	},
		[]string{
			"backrest_ver",
			"backup_name",
			"backup_type",
			"block_incr",
			"database_id",
			"lsn_start",
			"lsn_stop",
			"pg_version",
			"prior",
			"repo_key",
			"stanza",
			"wal_start",
			"wal_stop"})
	pgbrStanzaBackupDurationMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_duration_seconds",
		Help: "Backup duration.",
	},
		[]string{
			"backup_name",
			"backup_type",
			"database_id",
			"repo_key",
			"stanza",
			"start_time",
			"stop_time"})
	// The 'database size' for text pgBackRest output
	// (or "backup":"info":"size" for json pgBackRest output)
	// is the full uncompressed size of the database.
	pgbrStanzaBackupDatabaseSizeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_size_bytes",
		Help: "Full uncompressed size of the database.",
	},
		[]string{
			"backup_name",
			"backup_type",
			"database_id",
			"repo_key",
			"stanza"})
	// The 'database backup size' for text pgBackRest output
	// (or "backup":"info":"delta" for json pgBackRest output)
	// is the amount of data in the database
	// to actually backup (these will be the same for full backups).
	pgbrStanzaBackupDatabaseBackupSizeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_delta_bytes",
		Help: "Amount of data in the database to actually backup.",
	},
		[]string{
			"backup_name",
			"backup_type",
			"database_id",
			"repo_key",
			"stanza"})
	// The 'backup set size' for text pgBackRest output
	// (or "backup":"info":"repository":"size" for json pgBackRest output)
	// includes all the files from this backup and
	// any referenced backups in the repository that are required
	// to restore the database from this backup.
	// Repository 'backup set size' reflect compressed file sizes
	// if compression is enabled in pgBackRest.
	// Before pgBackRest v2.38 - reflect compressed file sizes
	// if compression is enabled in pgBackRest or filesystem.
	// From pgbackRest v2.38 the logic that tried
	// to determine additional file system compression was removed.
	pgbrStanzaBackupRepoBackupSetSizeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_repo_size_bytes",
		Help: "Full compressed files size to restore the database from backup.",
	},
		[]string{
			"backup_name",
			"backup_type",
			"database_id",
			"repo_key",
			"stanza"})
	// For json pgBackRest output
	// "backup":"info":"repository":"size-map"
	// Size of block incremental map (0 if no map).
	pgbrStanzaBackupRepoBackupSetSizeMapMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_repo_size_map_bytes",
		Help: "Size of block incremental map.",
	},
		[]string{
			"backup_name",
			"backup_type",
			"database_id",
			"repo_key",
			"stanza"})
	// The 'backup size' for text pgBackRest output
	// (or "backup":"info":"repository":"delta" for json pgBackRest output)
	// includes only the files in this backup
	// (these will also be the same for full backups).
	// Repository 'backup size' reflect compressed file sizes
	// if compression is enabled in pgBackRest.
	// Before pgBackRest v2.38 - reflect compressed file sizes
	// if compression is enabled in pgBackRest or filesystem.
	// From pgbackRest v2.38 the logic that tried
	// to determine additional file system compression was removed.
	pgbrStanzaBackupRepoBackupSizeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_repo_delta_bytes",
		Help: "Compressed files size in backup.",
	},
		[]string{
			"backup_name",
			"backup_type",
			"database_id",
			"repo_key",
			"stanza"})
	// For json pgBackRest output
	// "backup":"info":"repository":"delta-map"
	// Size of block incremental delta map if block incremental.
	pgbrStanzaBackupRepoBackupSizeMapMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_repo_delta_map_bytes",
		Help: "Size of block incremental delta map.",
	},
		[]string{
			"backup_name",
			"backup_type",
			"database_id",
			"repo_key",
			"stanza"})
	pgbrStanzaBackupErrorMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_error_status",
		Help: "Backup error status.",
	},
		[]string{
			"backup_name",
			"backup_type",
			"database_id",
			"repo_key",
			"stanza"})
	pgbrStanzaBackupDatabasesMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_databases",
		Help: "Number of databases in backup.",
	},
		[]string{
			"backup_name",
			"backup_type",
			"database_id",
			"repo_key",
			"stanza"})
)

// Set backup metrics:
//   - pgbackrest_backup_info
//   - pgbackrest_backup_duration_seconds
//   - pgbackrest_backup_size_bytes
//   - pgbackrest_backup_delta_bytes
//   - pgbackrest_backup_repo_size_bytes
//   - pgbackrest_backup_repo_size_map_bytes
//   - pgbackrest_backup_repo_delta_bytes
//   - pgbackrest_backup_repo_delta_map_bytes
//   - pgbackrest_backup_error_status
//   - pgbackrest_backup_databases
//
// And returns info about last backups.
func getBackupMetrics(config, configIncludePath, stanzaName string, backupData []backup, dbData []db, backupDBCount bool, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) lastBackupsStruct {
	var (
		err                     error
		parseStanzaDataSpecific []stanza
		blockIncr               string
	)
	lastBackups := lastBackupsStruct{}
	// Each backup for current stanza.
	for _, backup := range backupData {
		// For pgBackRest >= v2.44 the functionality to perform a block incremental backup has appeared.
		// The block size is determined based on the file size and age.
		// Very old or very small files will not use block incremental.
		// By default, the block incremental is disable for backups. See `--repo-block` option.
		blockIncr = "n"
		// Block incremental map is used for block level backup .
		// If one value from 'size-map' or 'delta-map' is nil, and other has correct value,
		// it looks like a bug in pgBackRest.
		// See https://github.com/pgbackrest/pgbackrest/blob/3feed389a2199454db68e446851323498b45db20/src/command/info/info.c#L459-L463
		// Relation - backupInfoRepoSizeMap != NULL, where backupInfoRepoSizeMap is related to SizeMap (size-map).
		if backup.Info.Repository.SizeMap != nil && backup.Info.Repository.DeltaMap != nil {
			// The block incremental backup functionality is used.
			blockIncr = "y"
			// Repo backup map size.
			setUpMetric(
				pgbrStanzaBackupRepoBackupSetSizeMapMetric,
				"pgbackrest_backup_repo_size_map_bytes",
				float64(*backup.Info.Repository.SizeMap),
				setUpMetricValueFun,
				logger,
				backup.Label,
				backup.Type,
				strconv.Itoa(backup.Database.ID),
				strconv.Itoa(backup.Database.RepoKey),
				stanzaName,
			)
			// Repo backup delta map size.
			setUpMetric(
				pgbrStanzaBackupRepoBackupSizeMapMetric,
				"pgbackrest_backup_repo_delta_map_bytes",
				float64(*backup.Info.Repository.DeltaMap),
				setUpMetricValueFun,
				logger,
				backup.Label,
				backup.Type,
				strconv.Itoa(backup.Database.ID),
				strconv.Itoa(backup.Database.RepoKey),
				stanzaName,
			)
		}
		//  1 - info about backup is exist.
		setUpMetric(
			pgbrStanzaBackupInfoMetric,
			"pgbackrest_backup_info",
			1,
			setUpMetricValueFun,
			logger,
			backup.BackrestInfo.Version,
			backup.Label,
			backup.Type,
			blockIncr,
			strconv.Itoa(backup.Database.ID),
			backup.Lsn.StartLSN,
			backup.Lsn.StopLSN,
			getPGVersion(backup.Database.ID, backup.Database.RepoKey, dbData),
			backup.Prior,
			strconv.Itoa(backup.Database.RepoKey),
			stanzaName,
			backup.Archive.StartWAL,
			backup.Archive.StopWAL,
		)
		// Backup durations in seconds.
		setUpMetric(
			pgbrStanzaBackupDurationMetric,
			"pgbackrest_backup_duration_seconds",
			time.Unix(backup.Timestamp.Stop, 0).Sub(time.Unix(backup.Timestamp.Start, 0)).Seconds(),
			setUpMetricValueFun,
			logger,
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			stanzaName,
			time.Unix(backup.Timestamp.Start, 0).Format(layout),
			time.Unix(backup.Timestamp.Stop, 0).Format(layout),
		)
		// Database size.
		setUpMetric(
			pgbrStanzaBackupDatabaseSizeMetric,
			"pgbackrest_backup_size_bytes",
			float64(backup.Info.Size),
			setUpMetricValueFun,
			logger,
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			stanzaName,
		)
		// Database backup size.
		setUpMetric(
			pgbrStanzaBackupDatabaseBackupSizeMetric,
			"pgbackrest_backup_delta_bytes",
			float64(backup.Info.Delta),
			setUpMetricValueFun,
			logger,
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			stanzaName,
		)
		// Repo backup set size.
		// Starting from pgBackRest v2.45, there is no 'backup set size' value
		// for block incremental backups.
		// See https://github.com/pgbackrest/pgbackrest/commit/6252c0e4485caee362edec13302a5f735a69bff4
		// and https://github.com/pgbackrest/pgbackrest/projects/2#card-87759001
		// This behavior may change in future pgBackRest releases.
		if backup.Info.Repository.Size != nil {
			setUpMetric(
				pgbrStanzaBackupRepoBackupSetSizeMetric,
				"pgbackrest_backup_repo_size_bytes",
				float64(*backup.Info.Repository.Size),
				setUpMetricValueFun,
				logger,
				backup.Label,
				backup.Type,
				strconv.Itoa(backup.Database.ID),
				strconv.Itoa(backup.Database.RepoKey),
				stanzaName,
			)
		}
		// Repo backup size.
		setUpMetric(
			pgbrStanzaBackupRepoBackupSizeMetric,
			"pgbackrest_backup_repo_delta_bytes",
			float64(backup.Info.Repository.Delta),
			setUpMetricValueFun,
			logger,
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			stanzaName,
		)
		// Backup error status.
		// Use *bool type for backup.Error field.
		// Information about error in backup (page checksum error) has appeared since pgBackRest v2.36.
		// In versions < v2.36 this field is missing and the metric does not need to be collected.
		// json.Unmarshal() will return nil when the error information is  missing.
		if backup.Error != nil {
			setUpMetric(
				pgbrStanzaBackupErrorMetric,
				"pgbackrest_backup_error_status",
				convertBoolToFloat64(*backup.Error),
				setUpMetricValueFun,
				logger,
				backup.Label,
				backup.Type,
				strconv.Itoa(backup.Database.ID),
				strconv.Itoa(backup.Database.RepoKey),
				stanzaName,
			)
		}
		// If the calculation of the number of databases in backups is enabled.
		// Information about number of databases in specific backup has appeared since pgBackRest v2.41.
		// In versions < v2.41 this is missing and the metric does not need to be collected.
		// getParsedSpecificBackupInfoData will return error in this case.
		if backupDBCount {
			// Try to get info for backup.
			parseStanzaDataSpecific, err = getParsedSpecificBackupInfoData(config, configIncludePath, stanzaName, backup.Label, logger)
			if err == nil {
				// In a normal situation, only one element with one backup should be returned.
				// If more than one element or one backup is returned, there is may be a bug in pgBackRest.
				// If it's not a bug, then this part will need to be refactoring.
				// Use *[]struct() type for backup.DatabaseRef.
				if parseStanzaDataSpecific[0].Backup[0].DatabaseRef != nil {
					// Number of databases in the last full backup.
					setUpMetric(
						pgbrStanzaBackupDatabasesMetric,
						"pgbackrest_backup_databases",
						float64(len(*parseStanzaDataSpecific[0].Backup[0].DatabaseRef)),
						setUpMetricValueFun,
						logger,
						backup.Label,
						backup.Type,
						strconv.Itoa(backup.Database.ID),
						strconv.Itoa(backup.Database.RepoKey),
						stanzaName,
					)
				}
			}
		}
		compareLastBackups(
			&lastBackups,
			time.Unix(backup.Timestamp.Stop, 0),
			backup.Label,
			backup.Type,
		)
	}
	return lastBackups
}

func resetBackupMetrics() {
	pgbrStanzaBackupInfoMetric.Reset()
	pgbrStanzaBackupDurationMetric.Reset()
	pgbrStanzaBackupDatabaseSizeMetric.Reset()
	pgbrStanzaBackupDatabaseBackupSizeMetric.Reset()
	pgbrStanzaBackupRepoBackupSetSizeMetric.Reset()
	pgbrStanzaBackupRepoBackupSetSizeMapMetric.Reset()
	pgbrStanzaBackupRepoBackupSizeMetric.Reset()
	pgbrStanzaBackupRepoBackupSizeMapMetric.Reset()
	pgbrStanzaBackupErrorMetric.Reset()
	pgbrStanzaBackupDatabasesMetric.Reset()
}
