package backrest

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	pgbrStanzaStatusMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_stanza_status",
		Help: "Current stanza status.",
	},
		[]string{"stanza"})
	pgbrRepoStatusMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_repo_status",
		Help: "Current repository status.",
	},
		[]string{
			"cipher",
			"repo_key",
			"stanza",
		})
	pgbrStanzaBackupInfoMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_info",
		Help: "Backup info.",
	},
		[]string{
			"backrest_ver",
			"backup_name",
			"backup_type",
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
	pgbrWALArchivingMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_wal_archive_status",
		Help: "Current WAL archive status.",
	},
		[]string{
			"database_id",
			"pg_version",
			"repo_key",
			"stanza",
			"wal_max",
			"wal_min"})
	pgbrExporterInfoMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_exporter_info",
		Help: "Information about pgBackRest exporter.",
	},
		[]string{"version"})
)
