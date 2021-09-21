package backrest

import (
	"encoding/json"
	"errors"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type setUpMetricValueFunType func(metric *prometheus.GaugeVec, value float64, labels ...string) error

type lastBackupsStruct struct {
	full time.Time
	diff time.Time
	incr time.Time
}

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
	execCommand = exec.Command
)

// https://golang.org/pkg/time/#Time.Format
const layout = "2006-01-02 15:04:05"

func returnDefaultExecArgs() []string {
	// Base exec arguments.
	defaultArgs := []string{"info", "--output", "json"}
	return defaultArgs
}

func returnConfigExecArgs(config, configIncludePath string) []string {
	var configArgs []string
	switch {
	case config == "" && configIncludePath == "":
		// Use default config and config-include-path (or define by env).
		configArgs = []string{}
	case config != "" && configIncludePath == "":
		// Use custom config.
		configArgs = []string{"--config", config}
	case config == "" && configIncludePath != "":
		// Use custom config-include-path.
		configArgs = []string{"--config-include-path", configIncludePath}
	default:
		// Use custom config and config-include-path.
		configArgs = []string{"--config", config, "--config-include-path", configIncludePath}
	}
	return configArgs
}

func returnConfigStanzaArgs(stanza string) []string {
	var stanzaArgs []string
	switch {
	case stanza == "":
		// Stanza not set. No return parametrs.
		stanzaArgs = []string{}
	default:
		// Use specific stanza.
		stanzaArgs = []string{"--stanza", stanza}
	}
	return stanzaArgs
}

func concatExecArgs(slices [][]string) []string {
	tmp := []string{}
	for _, s := range slices {
		tmp = append(tmp, s...)
	}
	return tmp
}

func getAllInfoData(config, configIncludePath, stanza string) ([]byte, error) {
	app := "pgbackrest"
	args := [][]string{
		returnDefaultExecArgs(),
		returnConfigExecArgs(config, configIncludePath),
		returnConfigStanzaArgs(stanza),
	}
	// Finally arguments for exec command
	concatArgs := concatExecArgs(args)
	out, err := execCommand(app, concatArgs...).CombinedOutput()
	// If error occurs - write error from pgBackRest to log and
	// return nil for stanza data.
	if err != nil {
		log.Printf("[ERROR] pgBackRest error: %v", string(out))
		return nil, err
	}
	return out, err
}

func parseResult(output []byte) ([]stanza, error) {
	err := json.Unmarshal(output, &stanzas)
	return stanzas, err
}

func getPGVersion(id, repoKey int, dbList []db) string {
	for _, db := range dbList {
		if id == db.ID && repoKey == db.RepoKey {
			return db.Version
		}
	}
	return ""
}

func getMetrics(data stanza, verbose bool, currentUnixTime int64, setUpMetricValueFun setUpMetricValueFunType) {
	var err error
	lastBackups := lastBackupsStruct{}
	//https://github.com/pgbackrest/pgbackrest/blob/03021c6a17f1374e84ef42614fa1dd2a6be4b64d/src/command/info/info.c#L78-L94
	// Stanza and repo statuses:
	//  0: "ok",
	//  1: "missing stanza path",
	//  2: "no valid backups",
	//  3: "missing stanza data",
	//  4: "different across repos",
	//  5: "database mismatch across repos",
	//  6: "requested backup not found",
	//  99: "other".
	err = setUpMetricValueFun(
		pgbrStanzaStatusMetric,
		float64(data.Status.Code),
		data.Name,
	)
	if err != nil {
		log.Println(
			"[ERROR] Metric pgbackrest_stanza_status set up failed;",
			"value:", float64(data.Status.Code), ";",
			"labels:",
			data.Name,
		)
	}
	// Repo status.
	for _, repo := range data.Repo {
		err = setUpMetricValueFun(
			pgbrRepoStatusMetric,
			float64(repo.Status.Code),
			repo.Cipher,
			strconv.Itoa(repo.Key),
			data.Name,
		)
		if err != nil {
			log.Println(
				"[ERROR] Metric pgbackrest_repo_status set up failed;",
				"value:", float64(repo.Status.Code), ";",
				"labels:",
				repo.Cipher, ",",
				strconv.Itoa(repo.Key), ",",
				data.Name,
			)
		}
	}
	// Each backup for current stanza.
	for _, backup := range data.Backup {
		//  1 - info about backup is exist.
		err = setUpMetricValueFun(
			pgbrStanzaBackupInfoMetric,
			1,
			backup.BackrestInfo.Version,
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			getPGVersion(backup.Database.ID, backup.Database.RepoKey, data.DB),
			backup.Prior,
			strconv.Itoa(backup.Database.RepoKey),
			data.Name,
			backup.Archive.StartWAL,
			backup.Archive.StopWAL,
		)
		if err != nil {
			log.Println(
				"[ERROR] Metric pgbackrest_backup_info set up failed;",
				"value:", 1, ";",
				"labels:",
				backup.BackrestInfo.Version, ",",
				backup.Label, ",",
				backup.Type, ",",
				strconv.Itoa(backup.Database.ID), ",",
				getPGVersion(backup.Database.ID, backup.Database.RepoKey, data.DB), ",",
				backup.Prior, ",",
				strconv.Itoa(backup.Database.RepoKey), ",",
				data.Name, ",",
				backup.Archive.StartWAL, ",",
				backup.Archive.StopWAL,
			)
		}
		// Backup durations in seconds.
		err = setUpMetricValueFun(
			pgbrStanzaBackupDurationMetric,
			time.Unix(backup.Timestamp.Stop, 0).Sub(time.Unix(backup.Timestamp.Start, 0)).Seconds(),
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			data.Name,
			time.Unix(backup.Timestamp.Start, 0).Format(layout),
			time.Unix(backup.Timestamp.Stop, 0).Format(layout),
		)
		if err != nil {
			log.Println(
				"[ERROR] Metric pgbackrest_backup_duration_seconds set up failed;",
				"value:",
				time.Unix(backup.Timestamp.Stop, 0).Sub(time.Unix(backup.Timestamp.Start, 0)).Seconds(),
				";",
				"labels:",
				backup.Label, ",",
				backup.Type, ",",
				strconv.Itoa(backup.Database.ID), ",",
				strconv.Itoa(backup.Database.RepoKey), ",",
				data.Name, ",",
				time.Unix(backup.Timestamp.Start, 0).Format(layout), ",",
				time.Unix(backup.Timestamp.Stop, 0).Format(layout),
			)
		}
		// Database size.
		err = setUpMetricValueFun(
			pgbrStanzaBackupDatabaseSizeMetric,
			float64(backup.Info.Size),
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			data.Name,
		)
		if err != nil {
			log.Println(
				"[ERROR] Metric pgbackrest_backup_size_bytes set up failed;",
				"value:", float64(backup.Info.Size), ";",
				"labels:",
				backup.Label, ",",
				backup.Type, ",",
				strconv.Itoa(backup.Database.ID), ",",
				strconv.Itoa(backup.Database.RepoKey), ",",
				data.Name,
			)
		}
		// Database backup size.
		err = setUpMetricValueFun(
			pgbrStanzaBackupDatabaseBackupSizeMetric,
			float64(backup.Info.Delta),
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			data.Name,
		)
		if err != nil {
			log.Println(
				"[ERROR] Metric pgbackrest_backup_delta_bytes set up failed;",
				"value:", float64(backup.Info.Delta), ";",
				"labels:",
				backup.Label, ",",
				backup.Type, ",",
				strconv.Itoa(backup.Database.ID), ",",
				strconv.Itoa(backup.Database.RepoKey), ",",
				data.Name,
			)
		}
		// Repo backup set size.
		err = setUpMetricValueFun(
			pgbrStanzaBackupRepoBackupSetSizeMetric,
			float64(backup.Info.Repository.Size),
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			data.Name,
		)
		if err != nil {
			log.Println(
				"[ERROR] Metric pgbackrest_backup_repo_size_bytes set up failed;",
				"value:", float64(backup.Info.Repository.Size), ";",
				"labels:",
				backup.Label, ",",
				backup.Type, ",",
				strconv.Itoa(backup.Database.ID), ",",
				strconv.Itoa(backup.Database.RepoKey), ",",
				data.Name,
			)
		}
		// Repo backup size.
		err = setUpMetricValueFun(
			pgbrStanzaBackupRepoBackupSizeMetric,
			float64(backup.Info.Repository.Delta),
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			data.Name,
		)
		if err != nil {
			log.Println(
				"[ERROR] Metric pgbackrest_backup_repo_delta_bytes set up failed;",
				"value:", float64(backup.Info.Repository.Delta), ";",
				"labels:",
				backup.Label, ",",
				backup.Type, ",",
				strconv.Itoa(backup.Database.ID), ",",
				strconv.Itoa(backup.Database.RepoKey), ",",
				data.Name,
			)
		}
		compareLastBackups(
			&lastBackups,
			time.Unix(backup.Timestamp.Stop, 0),
			backup.Type,
		)
	}
	// If full backup exists, the values of metrics for differential and
	// incremental backups also will be set.
	// If not - metrics won't be set.
	if !lastBackups.full.IsZero() {
		// Seconds since the last completed full backup.
		err = setUpMetricValueFun(
			pgbrStanzaBackupLastFullMetric,
			// Trim nanoseconds.
			time.Unix(currentUnixTime, 0).Sub(lastBackups.full).Seconds(),
			data.Name,
		)
		if err != nil {
			log.Println(
				"[ERROR] Metric pgbackrest_backup_full_since_last_completion_seconds set up failed;",
				"value:", time.Unix(currentUnixTime, 0).Sub(lastBackups.full).Seconds(), ";",
				"labels:",
				data.Name,
			)
		}
		// Seconds since the last completed full or differential backup.
		err = setUpMetricValueFun(
			pgbrStanzaBackupLastDiffMetric,
			time.Unix(currentUnixTime, 0).Sub(lastBackups.diff).Seconds(),
			data.Name,
		)
		if err != nil {
			log.Println(
				"[ERROR] Metric pgbackrest_backup_diff_since_last_completion_seconds set up failed;",
				"value:", time.Unix(currentUnixTime, 0).Sub(lastBackups.diff).Seconds(), ";",
				"labels:",
				data.Name,
			)
		}
		// Seconds since the last completed full, differential or incremental backup.
		err = setUpMetricValueFun(
			pgbrStanzaBackupLastIncrMetric,
			time.Unix(currentUnixTime, 0).Sub(lastBackups.incr).Seconds(),
			data.Name,
		)
		if err != nil {
			log.Println(
				"[ERROR] Metric pgbackrest_backup_incr_since_last_completion_seconds set up failed;",
				"value:", time.Unix(currentUnixTime, 0).Sub(lastBackups.incr).Seconds(), ";",
				"labels:",
				data.Name,
			)
		}
	}
	// WAL archive info.
	// 0 - any one of WALMin and WALMax have empty value, there is no correct information about WAL archiving.
	// 1 - both WALMin and WALMax have no empty values, there is correct information about WAL archiving.
	// Verbose mode.
	// When "verbose == true" - WALMin and WALMax are added as metric labels.
	// This creates new different time series on each WAL archiving which maybe is not right way.
	for _, archive := range data.Archive {
		if archive.WALMin != "" && archive.WALMax != "" {
			if verbose {
				err = setUpMetricValueFun(
					pgbrWALArchivingMetric,
					1,
					strconv.Itoa(archive.Database.ID),
					getPGVersion(archive.Database.ID, archive.Database.RepoKey, data.DB),
					strconv.Itoa(archive.Database.RepoKey),
					data.Name,
					archive.WALMax,
					archive.WALMin,
				)
				if err != nil {
					log.Println(
						"[ERROR] Metric pgbackrest_wal_archive_status set up failed;",
						"value:", 1, ";",
						"labels:",
						strconv.Itoa(archive.Database.ID), ",",
						getPGVersion(archive.Database.ID, archive.Database.RepoKey, data.DB), ",",
						strconv.Itoa(archive.Database.RepoKey), ",",
						data.Name, ",",
						archive.WALMax, ",",
						archive.WALMin,
					)
				}
			} else {
				err = setUpMetricValueFun(
					pgbrWALArchivingMetric,
					1,
					strconv.Itoa(archive.Database.ID),
					getPGVersion(archive.Database.ID, archive.Database.RepoKey, data.DB),
					strconv.Itoa(archive.Database.RepoKey),
					data.Name,
					"",
					"",
				)
				if err != nil {
					log.Println(
						"[ERROR] Metric pgbackrest_wal_archive_status set up failed;",
						"value:", 1, ";",
						"labels:",
						strconv.Itoa(archive.Database.ID), ",",
						getPGVersion(archive.Database.ID, archive.Database.RepoKey, data.DB), ",",
						strconv.Itoa(archive.Database.RepoKey), ",",
						data.Name, ",",
						"\"\"", ",",
						"\"\"",
					)
				}
			}
		} else {
			err = setUpMetricValueFun(
				pgbrWALArchivingMetric,
				0,
				strconv.Itoa(archive.Database.ID),
				getPGVersion(archive.Database.ID, archive.Database.RepoKey, data.DB),
				strconv.Itoa(archive.Database.RepoKey),
				data.Name,
				"",
				"",
			)
			if err != nil {
				log.Println(
					"[ERROR] Metric pgbackrest_wal_archive_status set up failed;",
					"value:", 0, ";",
					"labels:",
					strconv.Itoa(archive.Database.ID), ",",
					getPGVersion(archive.Database.ID, archive.Database.RepoKey, data.DB), ",",
					strconv.Itoa(archive.Database.RepoKey), ",",
					data.Name, ",",
					"\"\"", ",",
					"\"\"",
				)
			}
		}
	}
}

func setUpMetricValue(metric *prometheus.GaugeVec, value float64, labels ...string) error {
	metricVec, err := metric.GetMetricWithLabelValues(labels...)
	if err != nil {
		log.Printf("[ERROR] Metric initialization failed, %v", err)
		return err
	}
	// The situation should be handled by the prometheus libraries.
	// But, anything is possible.
	if metricVec == nil {
		err := errors.New("metric is nil")
		log.Printf("[ERROR] %v", err)
		return err
	}
	metricVec.Set(value)
	return nil
}

func compareLastBackups(backups *lastBackupsStruct, currentBackup time.Time, backupType string) {
	switch backupType {
	case "full":
		if currentBackup.After(backups.full) {
			backups.full = currentBackup
		}
		if currentBackup.After(backups.diff) {
			backups.diff = currentBackup
		}
		if currentBackup.After(backups.incr) {
			backups.incr = currentBackup
		}
	case "diff":
		if currentBackup.After(backups.diff) {
			backups.diff = currentBackup
		}
		if currentBackup.After(backups.incr) {
			backups.incr = currentBackup
		}
	case "incr":
		if currentBackup.After(backups.incr) {
			backups.incr = currentBackup
		}
	}
}

func stanzaNotInExclude(stanza string, listExclude []string) bool {
	// Сheck that exclude list is empty.
	// If so, no excluding stanzas are set during startup.
	if strings.Join(listExclude, "") != "" {
		for _, val := range listExclude {
			if val == stanza {
				return false
			}
		}
	}
	return true
}
