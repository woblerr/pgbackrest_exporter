package backrest

import (
	"net/http"
	"os"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/web"
)

var (
	promPort               string
	promEndpoint           string
	promTLSConfigPath      string
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
	// if compression is enabled in pgBackRest or the filesystem.
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
	// if compression is enabled in pgBackRest or the filesystem.
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
	pgbrStanzaBackupLastFullMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_full_since_last_completion_seconds",
		Help: "Seconds since the last completed full backup.",
	},
		[]string{"stanza"})
	// Differential backup is always based on last full,
	// if the last backup was full, the metric will take full backup value.
	pgbrStanzaBackupLastDiffMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_diff_since_last_completion_seconds",
		Help: "Seconds since the last completed full or differential backup.",
	},
		[]string{"stanza"})
	// Incremental backup is always based on last full or differential,
	// if the last backup was full or differential, the metric will take
	// full or differential backup value.
	pgbrStanzaBackupLastIncrMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_backup_incr_since_last_completion_seconds",
		Help: "Seconds since the last completed full, differential or incremental backup.",
	},
		[]string{"stanza"})
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

// SetPromPortandPath sets HTTP endpoint parameters
// from command line arguments 'port', 'endpoint' and 'tlsConfigPath'
func SetPromPortandPath(port, endpoint, tlsConfigPath string) {
	promPort = port
	promEndpoint = endpoint
	promTLSConfigPath = tlsConfigPath
}

// StartPromEndpoint run HTTP endpoint
func StartPromEndpoint(logger log.Logger) {
	go func(logger log.Logger) {
		http.Handle(promEndpoint, promhttp.Handler())
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<html>
			<head><title>pgBackRest exporter</title></head>
			<body>
			<h1>pgBackRest exporter</h1>
			<p><a href='` + promEndpoint + `'>Metrics</a></p>
			</body>
			</html>`))
		})
		server := &http.Server{Addr: ":" + promPort}
		if err := web.ListenAndServe(server, promTLSConfigPath, logger); err != nil {
			level.Error(logger).Log("msg", "Run web endpoint failed", "err", err)
			os.Exit(1)
		}
	}(logger)
}

// ResetMetrics reset metrics
func ResetMetrics() {
	pgbrStanzaStatusMetric.Reset()
	pgbrRepoStatusMetric.Reset()
	pgbrStanzaBackupInfoMetric.Reset()
	pgbrStanzaBackupDurationMetric.Reset()
	pgbrStanzaBackupDatabaseSizeMetric.Reset()
	pgbrStanzaBackupDatabaseBackupSizeMetric.Reset()
	pgbrStanzaBackupRepoBackupSetSizeMetric.Reset()
	pgbrStanzaBackupRepoBackupSizeMetric.Reset()
	pgbrStanzaBackupErrorMetric.Reset()
	pgbrStanzaBackupLastFullMetric.Reset()
	pgbrStanzaBackupLastDiffMetric.Reset()
	pgbrStanzaBackupLastIncrMetric.Reset()
	pgbrWALArchivingMetric.Reset()
}

// GetPgBackRestInfo get and parse pgBackRest info and set metrics
func GetPgBackRestInfo(config, configIncludePath string, stanzas []string, stanzasExclude []string, verbose bool, logger log.Logger) {
	// To calculate the time elapsed since the last completed full, differential or incremental backup.
	// For all stanzas values are calculated relative to one value.
	currentUnixTime := time.Now().Unix()
	// Loop over each stanza.
	// If stanza not set - perform a single loop step to get metrics for all stanzas.
	for _, stanza := range stanzas {
		// Check that stanza from the include list is not in the exclude list.
		// If stanza not set - checking for entry into the exclude list will be performed later.
		if stanzaNotInExclude(stanza, stanzasExclude) {
			stanzaData, err := getAllInfoData(config, configIncludePath, stanza, logger)
			if err != nil {
				level.Error(logger).Log("msg", "Get data from pgBackRest failed", "err", err)
			}
			parseStanzaData, err := parseResult(stanzaData)
			if err != nil {
				level.Error(logger).Log("msg", "Parse JSON failed", "err", err)
			}
			if len(parseStanzaData) == 0 {
				level.Warn(logger).Log("msg", "No backup data returned")
			}
			for _, singleStanza := range parseStanzaData {
				// If stanza is in the exclude list, skip it.
				if stanzaNotInExclude(singleStanza.Name, stanzasExclude) {
					getMetrics(singleStanza, verbose, currentUnixTime, setUpMetricValue, logger)
				}
			}
		} else {
			level.Warn(logger).Log("msg", "Stanza is specified in include and exclude lists", "stanza", stanza)
		}

	}
}

// GetExporterInfo set exporter info metric
func GetExporterInfo(exporterVersion string, logger log.Logger) {
	getExporterMetrics(exporterVersion, setUpMetricValue, logger)
}
