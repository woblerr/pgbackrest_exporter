package backrest

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

type setUpMetricValueFunType func(metric *prometheus.GaugeVec, value float64, labels ...string) error

type backupStruct struct {
	backupLabel string
	backupType  string
	backupTime  time.Time
}
type lastBackupsStruct struct {
	full backupStruct
	diff backupStruct
	incr backupStruct
}

var execCommand = exec.Command

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

func returnStanzaExecArgs(stanza string) []string {
	var stanzaArgs []string
	switch {
	case stanza == "":
		// Stanza not set. No return parameters.
		stanzaArgs = []string{}
	default:
		// Use specific stanza.
		stanzaArgs = []string{"--stanza", stanza}
	}
	return stanzaArgs
}

// Option 'type' cannot be set multiple times for info command.
// It's pgBackRest restriction.
func returnBackupTypeExecArgs(backupType string) []string {
	var backupTypeArgs []string
	switch {
	case backupType == "":
		// Backup type not set. No return parameters.
		backupTypeArgs = []string{}
	default:
		// Use specific backup type.
		backupTypeArgs = []string{"--type", backupType}
	}
	return backupTypeArgs
}

//
func returnBackupSetExecArgs(backupSetLabel string) []string {
	var backupSetLabelArgs []string
	switch {
	case backupSetLabel == "":
		// Backup label not set. No return parameters.
		backupSetLabelArgs = []string{}
	default:
		// Use specific backup label.
		backupSetLabelArgs = []string{"--set", backupSetLabel}
	}
	return backupSetLabelArgs
}

func concatExecArgs(slices [][]string) []string {
	tmp := []string{}
	for _, s := range slices {
		tmp = append(tmp, s...)
	}
	return tmp
}

func getAllInfoData(config, configIncludePath, stanza, backupType string, logger log.Logger) ([]byte, error) {
	var backupLabel string
	return getInfoData(config, configIncludePath, stanza, backupType, backupLabel, logger)
}

func getSpecificBackupInfoData(config, configIncludePath, stanza, backupLabel string, logger log.Logger) ([]byte, error) {
	var backupType string
	return getInfoData(config, configIncludePath, stanza, backupType, backupLabel, logger)
}

func getInfoData(config, configIncludePath, stanza, backupType, backupLabel string, logger log.Logger) ([]byte, error) {
	app := "pgbackrest"
	args := [][]string{
		returnDefaultExecArgs(),
		returnConfigExecArgs(config, configIncludePath),
		returnStanzaExecArgs(stanza),
		returnBackupTypeExecArgs(backupType),
	}
	if backupLabel != "" {
		args = append(args, returnBackupSetExecArgs(backupLabel))
	}
	// Finally arguments for exec command.
	concatArgs := concatExecArgs(args)
	cmd := execCommand(app, concatArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	// If stderr from pgBackRest is not empty,
	// write message from pgBackRest to log.
	if stderr.Len() > 0 {
		level.Error(logger).Log(
			"msg", "pgBackRest message",
			"err", stderr.String(),
		)
	}
	// If error occurs,
	// return nil for stanza data.
	if err != nil {
		return nil, err
	}
	return stdout.Bytes(), err
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

// Set stanza metrics:
//	* pgbackrest_stanza_status
func getStanzaMetrics(stanzaName string, stanzaStatusCode int, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) {
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
	level.Debug(logger).Log(
		"msg", "Metric pgbackrest_stanza_status",
		"value", float64(stanzaStatusCode),
		"labels", stanzaName,
	)
	err := setUpMetricValueFun(
		pgbrStanzaStatusMetric,
		float64(stanzaStatusCode),
		stanzaName,
	)
	if err != nil {
		level.Error(logger).Log(
			"msg", "Metric pgbackrest_stanza_status set up failed",
			"err", err,
		)
	}
}

// Set repo metrics:
//	* pgbackrest_repo_status
func getRepoMetrics(stanzaName string, repoData []repo, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) {
	// Repo status.
	// The same statuses as for stanza.
	for _, repo := range repoData {
		level.Debug(logger).Log(
			"msg", "Metric pgbackrest_repo_status",
			"value", float64(repo.Status.Code),
			"labels",
			strings.Join(
				[]string{
					repo.Cipher,
					strconv.Itoa(repo.Key),
					stanzaName,
				}, ",",
			),
		)
		err := setUpMetricValueFun(
			pgbrRepoStatusMetric,
			float64(repo.Status.Code),
			repo.Cipher,
			strconv.Itoa(repo.Key),
			stanzaName,
		)
		if err != nil {
			level.Error(logger).Log(
				"msg", "Metric pgbackrest_repo_status set up failed",
				"err", err,
			)
		}
	}
}

// Set backup metrics:
//	* pgbackrest_backup_info
//	* pgbackrest_backup_duration_seconds
//	* pgbackrest_backup_size_bytes
//	* pgbackrest_backup_delta_bytes
//	* pgbackrest_backup_repo_size_bytes
//	* pgbackrest_backup_repo_delta_bytes
//	* pgbackrest_backup_error_status
//	* pgbackrest_backup_databases
// And returns info about last backups.
func getBackupMetrics(config, configIncludePath, stanzaName string, backupData []backup, dbData []db, backupDBCount bool, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) lastBackupsStruct {
	var (
		err                     error
		parseStanzaDataSpecific []stanza
	)
	lastBackups := lastBackupsStruct{}
	// Each backup for current stanza.
	for _, backup := range backupData {
		//  1 - info about backup is exist.
		level.Debug(logger).Log(
			"msg", "Metric pgbackrest_backup_info",
			"value", 1,
			"labels",
			strings.Join(
				[]string{
					backup.BackrestInfo.Version,
					backup.Label,
					backup.Type,
					strconv.Itoa(backup.Database.ID),
					backup.Lsn.StartLSN,
					backup.Lsn.StopLSN,
					getPGVersion(backup.Database.ID, backup.Database.RepoKey, dbData),
					backup.Prior,
					strconv.Itoa(backup.Database.RepoKey),
					stanzaName,
					backup.Archive.StartWAL,
					backup.Archive.StopWAL,
				}, ",",
			),
		)
		err = setUpMetricValueFun(
			pgbrStanzaBackupInfoMetric,
			1,
			backup.BackrestInfo.Version,
			backup.Label,
			backup.Type,
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
		if err != nil {
			level.Error(logger).Log(
				"msg", "Metric pgbackrest_backup_info set up failed",
				"err", err,
			)
		}
		// Backup durations in seconds.
		level.Debug(logger).Log(
			"msg", "Metric pgbackrest_backup_duration_seconds",
			"value", time.Unix(backup.Timestamp.Stop, 0).Sub(time.Unix(backup.Timestamp.Start, 0)).Seconds(),
			"labels",
			strings.Join(
				[]string{
					backup.Label,
					backup.Type,
					strconv.Itoa(backup.Database.ID),
					strconv.Itoa(backup.Database.RepoKey),
					stanzaName,
					time.Unix(backup.Timestamp.Start, 0).Format(layout),
					time.Unix(backup.Timestamp.Stop, 0).Format(layout),
				}, ",",
			),
		)
		err = setUpMetricValueFun(
			pgbrStanzaBackupDurationMetric,
			time.Unix(backup.Timestamp.Stop, 0).Sub(time.Unix(backup.Timestamp.Start, 0)).Seconds(),
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			stanzaName,
			time.Unix(backup.Timestamp.Start, 0).Format(layout),
			time.Unix(backup.Timestamp.Stop, 0).Format(layout),
		)
		if err != nil {
			level.Error(logger).Log(
				"msg", "Metric pgbackrest_backup_duration_seconds set up failed",
				"err", err,
			)
		}
		// Database size.
		level.Debug(logger).Log(
			"msg", "Metric pgbackrest_backup_size_bytes",
			"value", float64(backup.Info.Size),
			"labels",
			strings.Join(
				[]string{
					backup.Label,
					backup.Type,
					strconv.Itoa(backup.Database.ID),
					strconv.Itoa(backup.Database.RepoKey),
					stanzaName,
				}, ",",
			),
		)
		err = setUpMetricValueFun(
			pgbrStanzaBackupDatabaseSizeMetric,
			float64(backup.Info.Size),
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			stanzaName,
		)
		if err != nil {
			level.Error(logger).Log(
				"msg", "Metric pgbackrest_backup_size_bytes set up failed",
				"err", err,
			)
		}
		// Database backup size.
		level.Debug(logger).Log(
			"msg", "Metric pgbackrest_backup_delta_bytes",
			"value", float64(backup.Info.Delta),
			"labels",
			strings.Join(
				[]string{
					backup.Label,
					backup.Type,
					strconv.Itoa(backup.Database.ID),
					strconv.Itoa(backup.Database.RepoKey),
					stanzaName,
				}, ",",
			),
		)
		err = setUpMetricValueFun(
			pgbrStanzaBackupDatabaseBackupSizeMetric,
			float64(backup.Info.Delta),
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			stanzaName,
		)
		if err != nil {
			level.Error(logger).Log(
				"msg", "Metric pgbackrest_backup_delta_bytes set up failed",
				"err", err,
			)
		}
		// Repo backup set size.
		level.Debug(logger).Log(
			"msg", "Metric pgbackrest_backup_repo_size_bytes",
			"value", float64(backup.Info.Repository.Size),
			"labels",
			strings.Join(
				[]string{
					backup.Label,
					backup.Type,
					strconv.Itoa(backup.Database.ID),
					strconv.Itoa(backup.Database.RepoKey),
					stanzaName,
				}, ",",
			),
		)
		err = setUpMetricValueFun(
			pgbrStanzaBackupRepoBackupSetSizeMetric,
			float64(backup.Info.Repository.Size),
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			stanzaName,
		)
		if err != nil {
			level.Error(logger).Log(
				"msg", "Metric pgbackrest_backup_repo_size_bytes set up failed",
				"err", err,
			)
		}
		// Repo backup size.
		level.Debug(logger).Log(
			"msg", "Metric pgbackrest_backup_repo_delta_bytes",
			"value", float64(backup.Info.Repository.Delta),
			"labels",
			strings.Join(
				[]string{
					backup.Label,
					backup.Type,
					strconv.Itoa(backup.Database.ID),
					strconv.Itoa(backup.Database.RepoKey),
					stanzaName,
				}, ",",
			),
		)
		err = setUpMetricValueFun(
			pgbrStanzaBackupRepoBackupSizeMetric,
			float64(backup.Info.Repository.Delta),
			backup.Label,
			backup.Type,
			strconv.Itoa(backup.Database.ID),
			strconv.Itoa(backup.Database.RepoKey),
			stanzaName,
		)
		if err != nil {
			level.Error(logger).Log(
				"msg", "Metric pgbackrest_backup_repo_delta_bytes set up failed",
				"err", err,
			)
		}
		// Backup error status.
		// Use *bool type for backup.Error field.
		// Information about error in backup (page checksum error) has appeared since pgBackRest v2.36.
		// In versions < v2.36 this field is missing and the metric does not need to be collected.
		// json.Unmarshal() will return nil when the error information is  missing.
		if backup.Error != nil {
			level.Debug(logger).Log(
				"msg", "Metric pgbackrest_backup_error_status",
				"value", convertBoolToFloat64(*backup.Error),
				"labels",
				strings.Join(
					[]string{
						backup.Label,
						backup.Type,
						strconv.Itoa(backup.Database.ID),
						strconv.Itoa(backup.Database.RepoKey),
						stanzaName,
					}, ",",
				),
			)
			err = setUpMetricValueFun(
				pgbrStanzaBackupErrorMetric,
				convertBoolToFloat64(*backup.Error),
				backup.Label,
				backup.Type,
				strconv.Itoa(backup.Database.ID),
				strconv.Itoa(backup.Database.RepoKey),
				stanzaName,
			)
			if err != nil {
				level.Error(logger).Log(
					"msg", "Metric pgbackrest_backup_error_status set up failed",
					"err", err,
				)
			}
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
					level.Debug(logger).Log(
						"msg", "Metric pgbackrest_backup_databases",
						"value", len(*parseStanzaDataSpecific[0].Backup[0].DatabaseRef),
						"labels",
						strings.Join(
							[]string{
								backup.Label,
								backup.Type,
								strconv.Itoa(backup.Database.ID),
								strconv.Itoa(backup.Database.RepoKey),
								stanzaName,
							}, ",",
						),
					)
					err = setUpMetricValueFun(
						pgbrStanzaBackupDatabasesMetric,
						float64(len(*parseStanzaDataSpecific[0].Backup[0].DatabaseRef)),
						backup.Label,
						backup.Type,
						strconv.Itoa(backup.Database.ID),
						strconv.Itoa(backup.Database.RepoKey),
						stanzaName,
					)
					if err != nil {
						level.Error(logger).Log(
							"msg", "Metric pgbackrest_backup_databases set up failed",
							"err", err,
						)
					}
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

// Set backup metrics:
//	* pgbackrest_backup_since_last_completion_seconds
//	* pgbackrest_backup_last_databases
func getBackupLastMetrics(config, configIncludePath, stanzaName string, lastBackups lastBackupsStruct, backupDBCountLatest bool, currentUnixTime int64, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) {
	var (
		err, errParse           error
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
		level.Debug(logger).Log(
			"msg", "Metric pgbackrest_backup_since_last_completion_seconds",
			"value", time.Unix(currentUnixTime, 0).Sub(lastBackups.full.backupTime).Seconds(),
			"labels",
			strings.Join(
				[]string{
					lastBackups.full.backupType,
					stanzaName,
				}, ",",
			),
		)
		err = setUpMetricValueFun(
			pgbrStanzaBackupSinceLastCompletionSecondsMetric,
			// Trim nanoseconds.
			time.Unix(currentUnixTime, 0).Sub(lastBackups.full.backupTime).Seconds(),
			lastBackups.full.backupType,
			stanzaName,
		)
		if err != nil {
			level.Error(logger).Log(
				"msg", "Metric pgbackrest_backup_since_last_completion_seconds set up failed",
				"err", err,
			)
		}
		// Seconds since the last completed full or differential backup.
		level.Debug(logger).Log(
			"msg", "Metric pgbackrest_backup_since_last_completion_seconds",
			"value", time.Unix(currentUnixTime, 0).Sub(lastBackups.diff.backupTime).Seconds(),
			"labels",
			strings.Join(
				[]string{
					lastBackups.diff.backupType,
					stanzaName,
				}, ",",
			),
		)
		err = setUpMetricValueFun(
			pgbrStanzaBackupSinceLastCompletionSecondsMetric,
			time.Unix(currentUnixTime, 0).Sub(lastBackups.diff.backupTime).Seconds(),
			lastBackups.diff.backupType,
			stanzaName,
		)
		if err != nil {
			level.Error(logger).Log(
				"msg", "Metric pgbackrest_backup_since_last_completion_seconds set up failed",
				"err", err,
			)
		}
		// Seconds since the last completed full, differential or incremental backup.
		level.Debug(logger).Log(
			"msg", "Metric pgbackrest_backup_since_last_completion_seconds",
			"value", time.Unix(currentUnixTime, 0).Sub(lastBackups.incr.backupTime).Seconds(),
			"labels",
			strings.Join(
				[]string{
					lastBackups.incr.backupType,
					stanzaName,
				}, ",",
			),
		)
		err = setUpMetricValueFun(
			pgbrStanzaBackupSinceLastCompletionSecondsMetric,
			time.Unix(currentUnixTime, 0).Sub(lastBackups.incr.backupTime).Seconds(),
			lastBackups.incr.backupType,
			stanzaName,
		)
		if err != nil {
			level.Error(logger).Log(
				"msg", "Metric pgbackrest_backup_since_last_completion_seconds set up failed",
				"err", err,
			)
		}
		// If the calculation of the number of databases in latest backups is enabled.
		// Information about number of databases in specific backup has appeared since pgBackRest v2.41.
		// In versions < v2.41 this is missing and the metric does not need to be collected.
		// getParsedSpecificBackupInfoData will return error in this case.
		if backupDBCountLatest {
			// Try to get info for full backup.
			parseStanzaDataSpecific, errParse = getParsedSpecificBackupInfoData(config, configIncludePath, stanzaName, lastBackups.full.backupLabel, logger)
			if errParse == nil {
				// In a normal situation, only one element with one backup should be returned.
				// If more than one element or one backup is returned, there is may be a bug in pgBackRest.
				// If it's not a bug, then this part will need to be refactoring.
				// Use *[]struct() type for backup.DatabaseRef.
				if parseStanzaDataSpecific[0].Backup[0].DatabaseRef != nil {
					// Number of databases in the last full backup.
					level.Debug(logger).Log(
						"msg", "Metric pgbackrest_backup_last_databases",
						"value", len(*parseStanzaDataSpecific[0].Backup[0].DatabaseRef),
						"labels",
						strings.Join(
							[]string{
								lastBackups.full.backupType,
								stanzaName,
							}, ",",
						),
					)
					err = setUpMetricValueFun(
						pgbrStanzaBackupLastDatabasesMetric,
						float64(len(*parseStanzaDataSpecific[0].Backup[0].DatabaseRef)),
						lastBackups.full.backupType,
						stanzaName,
					)
					if err != nil {
						level.Error(logger).Log(
							"msg", "Metric pgbackrest_backup_last_databases set up failed",
							"err", err,
						)
					}
				}
			}
			// If name for diff backup is equal to full, there is no point in re-receiving data.
			if lastBackups.diff.backupLabel != lastBackups.full.backupLabel {
				parseStanzaDataSpecific, errParse = getParsedSpecificBackupInfoData(config, configIncludePath, stanzaName, lastBackups.diff.backupLabel, logger)
			}
			if errParse == nil {
				if parseStanzaDataSpecific[0].Backup[0].DatabaseRef != nil {
					// Number of databases in the last full or differential backup.
					level.Debug(logger).Log(
						"msg", "Metric pgbackrest_backup_last_databases",
						"value", len(*parseStanzaDataSpecific[0].Backup[0].DatabaseRef),
						"labels",
						strings.Join(
							[]string{
								lastBackups.diff.backupType,
								stanzaName,
							}, ",",
						),
					)
					err = setUpMetricValueFun(
						pgbrStanzaBackupLastDatabasesMetric,
						float64(len(*parseStanzaDataSpecific[0].Backup[0].DatabaseRef)),
						lastBackups.diff.backupType,
						stanzaName,
					)
					if err != nil {
						level.Error(logger).Log(
							"msg", "Metric pgbackrest_backup_last_databases set up failed",
							"err", err,
						)
					}
				}
			}
			// If name for incr backup is equal to diff, there is no point in re-receiving data.
			if lastBackups.incr.backupLabel != lastBackups.diff.backupLabel {
				parseStanzaDataSpecific, errParse = getParsedSpecificBackupInfoData(config, configIncludePath, stanzaName, lastBackups.incr.backupLabel, logger)
			}
			if errParse == nil {
				if parseStanzaDataSpecific[0].Backup[0].DatabaseRef != nil {
					// Number of databases in the last full, differential or incremental backup.
					level.Debug(logger).Log(
						"msg", "Metric pgbackrest_backup_last_databases",
						"value", len(*parseStanzaDataSpecific[0].Backup[0].DatabaseRef),
						"labels",
						strings.Join(
							[]string{
								lastBackups.incr.backupType,
								stanzaName,
							}, ",",
						),
					)
					err = setUpMetricValueFun(
						pgbrStanzaBackupLastDatabasesMetric,
						float64(len(*parseStanzaDataSpecific[0].Backup[0].DatabaseRef)),
						lastBackups.incr.backupType,
						stanzaName,
					)
					if err != nil {
						level.Error(logger).Log(
							"msg", "Metric pgbackrest_backup_last_databases set up failed",
							"err", err,
						)
					}
				}
			}
		}
	}
}

// Set backup metrics:
//	* pgbackrest_wal_archive_status
func getWALMetrics(stanzaName string, archiveData []archive, dbData []db, verboseWAL bool, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) {
	var err error
	// WAL archive info.
	// 0 - any one of WALMin and WALMax have empty value, there is no correct information about WAL archiving.
	// 1 - both WALMin and WALMax have no empty values, there is correct information about WAL archiving.
	// Verbose mode.
	// When "verboseWAL == true" - WALMin and WALMax are added as metric labels.
	// This creates new different time series on each WAL archiving which maybe is not right way.
	for _, archive := range archiveData {
		if archive.WALMin != "" && archive.WALMax != "" {
			if verboseWAL {
				level.Debug(logger).Log(
					"msg", "Metric pgbackrest_wal_archive_status",
					"value", 1,
					"labels",
					strings.Join(
						[]string{
							strconv.Itoa(archive.Database.ID),
							getPGVersion(archive.Database.ID, archive.Database.RepoKey, dbData),
							strconv.Itoa(archive.Database.RepoKey),
							stanzaName,
							archive.WALMax,
							archive.WALMin,
						}, ",",
					),
				)
				err = setUpMetricValueFun(
					pgbrWALArchivingMetric,
					1,
					strconv.Itoa(archive.Database.ID),
					getPGVersion(archive.Database.ID, archive.Database.RepoKey, dbData),
					strconv.Itoa(archive.Database.RepoKey),
					stanzaName,
					archive.WALMax,
					archive.WALMin,
				)
				if err != nil {
					level.Error(logger).Log(
						"msg", "Metric pgbackrest_wal_archive_status set up failed",
						"err", err,
					)
				}
			} else {
				level.Debug(logger).Log(
					"msg", "Metric pgbackrest_wal_archive_status",
					"value", 1,
					"labels",
					strings.Join(
						[]string{
							strconv.Itoa(archive.Database.ID),
							getPGVersion(archive.Database.ID, archive.Database.RepoKey, dbData),
							strconv.Itoa(archive.Database.RepoKey),
							stanzaName,
							"''",
							"''",
						}, ",",
					),
				)
				err = setUpMetricValueFun(
					pgbrWALArchivingMetric,
					1,
					strconv.Itoa(archive.Database.ID),
					getPGVersion(archive.Database.ID, archive.Database.RepoKey, dbData),
					strconv.Itoa(archive.Database.RepoKey),
					stanzaName,
					"",
					"",
				)
				if err != nil {
					level.Error(logger).Log(
						"msg", "Metric pgbackrest_wal_archive_status set up failed",
						"err", err,
					)
				}
			}
		} else {
			level.Debug(logger).Log(
				"msg", "Metric pgbackrest_wal_archive_status",
				"value", 0,
				"labels",
				strings.Join(
					[]string{
						strconv.Itoa(archive.Database.ID),
						getPGVersion(archive.Database.ID, archive.Database.RepoKey, dbData),
						strconv.Itoa(archive.Database.RepoKey),
						stanzaName,
						"''",
						"''",
					}, ",",
				),
			)
			err = setUpMetricValueFun(
				pgbrWALArchivingMetric,
				0,
				strconv.Itoa(archive.Database.ID),
				getPGVersion(archive.Database.ID, archive.Database.RepoKey, dbData),
				strconv.Itoa(archive.Database.RepoKey),
				stanzaName,
				"",
				"",
			)
			if err != nil {
				level.Error(logger).Log(
					"msg", "Metric pgbackrest_wal_archive_status set up failed",
					"err", err,
				)
			}
		}
	}
}

func setUpMetricValue(metric *prometheus.GaugeVec, value float64, labels ...string) error {
	metricVec, err := metric.GetMetricWithLabelValues(labels...)
	if err != nil {
		return err
	}
	// The situation should be handled by the prometheus libraries.
	// But, anything is possible.
	if metricVec == nil {
		err := errors.New("metric is nil")
		return err
	}
	metricVec.Set(value)
	return nil
}

func compareLastBackups(backups *lastBackupsStruct, currentBackupTime time.Time, currentBackupLabel, currentBackupType string) {
	switch currentBackupType {
	case "full":
		if currentBackupTime.After(backups.full.backupTime) {
			backups.full.backupTime = currentBackupTime
			backups.full.backupLabel = currentBackupLabel
		}
		if currentBackupTime.After(backups.diff.backupTime) {
			backups.diff.backupTime = currentBackupTime
			backups.diff.backupLabel = currentBackupLabel
		}
		if currentBackupTime.After(backups.incr.backupTime) {
			backups.incr.backupTime = currentBackupTime
			backups.incr.backupLabel = currentBackupLabel
		}
	case "diff":
		if currentBackupTime.After(backups.diff.backupTime) {
			backups.diff.backupTime = currentBackupTime
			backups.diff.backupLabel = currentBackupLabel
		}
		if currentBackupTime.After(backups.incr.backupTime) {
			backups.incr.backupTime = currentBackupTime
			backups.incr.backupLabel = currentBackupLabel
		}
	case "incr":
		if currentBackupTime.After(backups.incr.backupTime) {
			backups.incr.backupTime = currentBackupTime
			backups.incr.backupLabel = currentBackupLabel
		}
	}
}

func stanzaNotInExclude(stanza string, listExclude []string) bool {
	// Check that exclude list is empty.
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

func getExporterMetrics(exporterVer string, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) {
	level.Debug(logger).Log(
		"msg", "Metric pgbackrest_exporter_info",
		"value", 1,
		"labels", exporterVer,
	)
	err := setUpMetricValueFun(
		pgbrExporterInfoMetric,
		1,
		exporterVer,
	)
	if err != nil {
		level.Error(logger).Log(
			"msg", "Metric pgbackrest_exporter_info set up failed",
			"err", err,
		)
	}
}

// Convert bool to float64.
func convertBoolToFloat64(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func getParsedSpecificBackupInfoData(config, configIncludePath, stanzaName, backupLabel string, logger log.Logger) ([]stanza, error) {
	stanzaDataSpecific, err := getSpecificBackupInfoData(config, configIncludePath, stanzaName, backupLabel, logger)
	if err != nil {
		level.Error(logger).Log(
			"msg", "Get data from pgBackRest failed",
			"stanza", stanzaName,
			"backup", backupLabel,
			"err", err)
	}
	parseDataSpecific, err := parseResult(stanzaDataSpecific)
	if err != nil {
		level.Error(logger).Log(
			"msg", "Parse JSON failed",
			"stanza", stanzaName,
			"backup", backupLabel,
			"err", err)
	}
	return parseDataSpecific, err
}
