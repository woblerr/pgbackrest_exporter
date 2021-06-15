package backrest

import (
	"encoding/json"
	"errors"
	"log"
	"os/exec"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	pgbrStanzaStatusMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_exporter_stanza_status",
		Help: "Current stanza status.",
	},
		[]string{"stanza"})
	pgbrRepoStatusMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_exporter_repo_status",
		Help: "Current repo status by stanza.",
	},
		[]string{
			"cipher",
			"repo_key",
			"stanza",
		})
	pgbrStanzaBackupInfoMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_exporter_backup_info",
		Help: "Backup info by stanza and backup type.",
	},
		[]string{
			"backrest_ver",
			"backup_name",
			"backup_type",
			"database_id",
			"pg_version",
			"repo_key",
			"stanza",
			"prior",
			"wal_archive_min",
			"wal_archive_max"})
	pgbrStanzaBackupDurationMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_exporter_backup_duration",
		Help: "Backup duration in seconds by stanza and backup type.",
	},
		[]string{
			"backup_name",
			"backup_type",
			"database_id",
			"repo_key",
			"stanza",
			"start_time",
			"stop_time"})
	pgbrStanzaBackupSizeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_exporter_backup_size",
		Help: "Backup size by stanza and backup type.",
	},
		[]string{
			"backup_name",
			"backup_type",
			"database_id",
			"repo_key",
			"stanza"})
	pgbrStanzaBackupDatabaseSizeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_exporter_backup_database_size",
		Help: "Database size in backup by stanza and backup type.",
	},
		[]string{
			"backup_name",
			"backup_type",
			"database_id",
			"repo_key",
			"stanza"})
	pgbrStanzaRepoBackupSetSizeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_exporter_backup_repo_backup_set_size",
		Help: "Repo set size in backup by stanza and backup type.",
	},
		[]string{
			"backup_name",
			"backup_type",
			"database_id",
			"repo_key",
			"stanza"})
	pgbrStanzaRepoBackupSizeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_exporter_backup_repo_backup_size",
		Help: "Repo size in backup by stanza and backup type.",
	},
		[]string{
			"backup_name",
			"backup_type",
			"database_id",
			"repo_key",
			"stanza"})
	pgbrWALArchivingMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_exporter_wal_archive_status",
		Help: "WAL archive status by stanza.",
	},
		[]string{
			"database_id",
			"pg_version",
			"repo_key",
			"stanza",
			"wal_archive_min",
			"wal_archive_max"})
	execCommand = exec.Command
)

// https://golang.org/pkg/time/#Time.Format
const layout = "2006-01-02 15:04:05"

func getAllInfoData() ([]byte, error) {
	out, err := execCommand("pgbackrest", "info", "--output", "json").Output()
	return out, err
}

func parseResult(output []byte) ([]stanza, error) {
	err := json.Unmarshal([]byte(output), &stanzas)
	return stanzas, err
}

func resetMetrics() {
	pgbrStanzaStatusMetric.Reset()
	pgbrRepoStatusMetric.Reset()
	pgbrStanzaBackupInfoMetric.Reset()
	pgbrStanzaBackupDurationMetric.Reset()
	pgbrStanzaBackupSizeMetric.Reset()
	pgbrStanzaBackupDatabaseSizeMetric.Reset()
	pgbrStanzaRepoBackupSetSizeMetric.Reset()
	pgbrStanzaRepoBackupSizeMetric.Reset()
	pgbrWALArchivingMetric.Reset()
}

func getPGVersion(id, repo_key int, db_list []db) string {
	for _, db := range db_list {
		if id == db.ID && repo_key == db.RepoKey {
			return db.Version
		}
	}
	return ""
}

func getMetrics(data stanza, verbose bool) {
	var err error
	// https://github.com/pgbackrest/pgbackrest/blob/master/src/command/info.c#L78-L94
	// Stanza and repo statuses:
	//  0: "ok",
	//  1: "missing stanza path",
	//  2: "no valid backups",
	//  3: "missing stanza data",
	//  4: "different across repos",
	//  5: "database mismatch across repos",
	//  6: "requested backup not found",
	//  99: "other".
	err = setUpMetricValue(
		pgbrStanzaStatusMetric,
		float64(data.Status.Code),
		data.Name,
	)
	if err != nil {
		log.Println(
			"[ERROR] Metric pgbackrest_exporter_stanza_status set up failed;",
			"value:", float64(data.Status.Code), ";",
			"labels:",
			data.Name, ".",
		)
	}
	// Each backup for current stanza.
	for _, backup := range data.Backup {
		//  1 - info about backup is exist.
		err = setUpMetricValue(
			pgbrStanzaBackupInfoMetric,
			1,
			backup.BackrestInfo.Version,
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			getPGVersion(backup.Database.ID, backup.Database.RepoKey, data.DB),
			strconv.Itoa(backup.Database.RepoKey),
			data.Name,
			backup.Prior,
			backup.Archive.StartWAL,
			backup.Archive.StopWAL,
		)
		if err != nil {
			log.Println(
				"[ERROR] Metric pgbackrest_exporter_backup_info set up failed;",
				"value:", 1, ";",
				"labels:",
				backup.BackrestInfo.Version, ",",
				backup.Label, ",",
				backup.Type, ",",
				strconv.Itoa(backup.Database.ID), ",",
				getPGVersion(backup.Database.ID, backup.Database.RepoKey, data.DB), ",",
				strconv.Itoa(backup.Database.RepoKey), ",",
				data.Name, ",",
				backup.Prior, ",",
				backup.Archive.StartWAL, ",",
				backup.Archive.StopWAL, ".",
			)
		}
		// Backup durations in seconds.
		err = setUpMetricValue(
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
				"[ERROR] Metric pgbackrest_exporter_backup_duration set up failed;",
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
				time.Unix(backup.Timestamp.Stop, 0).Format(layout), ".",
			)
		}
		// Backup size.
		err = setUpMetricValue(
			pgbrStanzaBackupSizeMetric,
			float64(backup.Info.Delta),
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			data.Name,
		)
		if err != nil {
			log.Println(
				"[ERROR] Metric pgbackrest_exporter_backup_size set up failed;",
				"value:", float64(backup.Info.Delta), ";",
				"labels:",
				backup.Label, ",",
				backup.Type, ",",
				strconv.Itoa(backup.Database.ID), ",",
				strconv.Itoa(backup.Database.RepoKey), ",",
				data.Name, ".",
			)
		}
		// Database size.
		err = setUpMetricValue(
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
				"[ERROR] Metric pgbackrest_exporter_backup_database_size set up failed;",
				"value:", float64(backup.Info.Size), ";",
				"labels:",
				backup.Label, ",",
				backup.Type, ",",
				strconv.Itoa(backup.Database.ID), ",",
				strconv.Itoa(backup.Database.RepoKey), ",",
				data.Name, ".",
			)
		}
		// Repo set size.
		err = setUpMetricValue(
			pgbrStanzaRepoBackupSetSizeMetric,
			float64(backup.Info.Repository.Size),
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			data.Name,
		)
		if err != nil {
			log.Println(
				"[ERROR] Metric pgbackrest_exporter_backup_repo_backup_set_size set up failed;",
				"value:", float64(backup.Info.Repository.Size), ";",
				"labels:",
				backup.Label, ",",
				backup.Type, ",",
				strconv.Itoa(backup.Database.ID), ",",
				strconv.Itoa(backup.Database.RepoKey), ",",
				data.Name, ".",
			)
		}
		// Repo size.
		err = setUpMetricValue(
			pgbrStanzaRepoBackupSizeMetric,
			float64(backup.Info.Repository.Delta),
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			data.Name,
		)
		if err != nil {
			log.Println(
				"[ERROR] Metric pgbackrest_exporter_backup_repo_backup_size set up failed;",
				"value:", float64(backup.Info.Repository.Delta), ";",
				"labels:",
				backup.Label, ",",
				backup.Type, ",",
				strconv.Itoa(backup.Database.ID), ",",
				strconv.Itoa(backup.Database.RepoKey), ",",
				data.Name, ".",
			)
		}
	}
	// Repo status.
	for _, repo := range data.Repo {
		err = setUpMetricValue(
			pgbrRepoStatusMetric,
			float64(repo.Status.Code),
			repo.Cipher,
			strconv.Itoa(repo.Key),
			data.Name,
		)
		if err != nil {
			log.Println(
				"[ERROR] Metric pgbackrest_exporter_repo_status set up failed;",
				"value:", float64(repo.Status.Code), ";",
				"labels:",
				repo.Cipher, ",",
				strconv.Itoa(repo.Key), ",",
				data.Name, ".",
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
				err = setUpMetricValue(
					pgbrWALArchivingMetric,
					1,
					strconv.Itoa(archive.Database.ID),
					getPGVersion(archive.Database.ID, archive.Database.RepoKey, data.DB),
					strconv.Itoa(archive.Database.RepoKey),
					data.Name,
					archive.WALMin,
					archive.WALMax,
				)
				if err != nil {
					log.Println(
						"[ERROR] Metric pgbackrest_exporter_wal_archive_status set up failed;",
						"value:", 1, ";",
						"labels:",
						strconv.Itoa(archive.Database.ID), ",",
						getPGVersion(archive.Database.ID, archive.Database.RepoKey, data.DB), ",",
						strconv.Itoa(archive.Database.RepoKey), ",",
						data.Name, ",",
						archive.WALMin, ",",
						archive.WALMax, ".",
					)
				}
			} else {
				err = setUpMetricValue(
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
						"[ERROR] Metric pgbackrest_exporter_wal_archive_status set up failed;",
						"value:", 1, ";",
						"labels:",
						strconv.Itoa(archive.Database.ID), ",",
						getPGVersion(archive.Database.ID, archive.Database.RepoKey, data.DB), ",",
						strconv.Itoa(archive.Database.RepoKey), ",",
						data.Name, ",",
						"", ",",
						"", ".",
					)
				}
			}
		} else {
			err = setUpMetricValue(
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
					"[ERROR] Metric pgbackrest_exporter_wal_archive_status set up failed;",
					"value:", 0, ";",
					"labels:",
					strconv.Itoa(archive.Database.ID), ",",
					getPGVersion(archive.Database.ID, archive.Database.RepoKey, data.DB), ",",
					strconv.Itoa(archive.Database.RepoKey), ",",
					data.Name, ",",
					"", ",",
					"", ".",
				)
			}
		}
	}
}

func setUpMetricValue(metric *prometheus.GaugeVec, value float64, labels ...string) error {
	metricVec, err := metric.GetMetricWithLabelValues(labels...)
	if err != nil {
		log.Printf("[ERROR] Metric initialization failed, %v.", err)
		return err
	}
	// The situation should be handled by the prometheus libraries.
	// But, anything is possible.
	if metricVec == nil {
		err := errors.New("metric is nil")
		log.Printf("[ERROR] %v.", err)
		return err
	}
	metricVec.Set(value)
	return nil
}
