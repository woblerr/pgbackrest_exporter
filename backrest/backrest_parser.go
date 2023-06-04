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
//   - pgbackrest_stanza_status
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
	setUpMetric(
		pgbrStanzaStatusMetric,
		"pgbackrest_stanza_status",
		float64(stanzaStatusCode),
		setUpMetricValueFun,
		logger,
		stanzaName,
	)
}

// Set repo metrics:
//   - pgbackrest_repo_status
func getRepoMetrics(stanzaName string, repoData []repo, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) {
	// Repo status.
	// The same statuses as for stanza.
	for _, repo := range repoData {
		setUpMetric(
			pgbrRepoStatusMetric,
			"pgbackrest_repo_status",
			float64(repo.Status.Code),
			setUpMetricValueFun,
			logger,
			repo.Cipher,
			strconv.Itoa(repo.Key),
			stanzaName,
		)
	}
}

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

// Set backup metrics:
//   - pgbackrest_wal_archive_status
func getWALMetrics(stanzaName string, archiveData []archive, dbData []db, verboseWAL bool, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) {
	// WAL archive info.
	// 0 - any one of WALMin and WALMax have empty value, there is no correct information about WAL archiving.
	// 1 - both WALMin and WALMax have no empty values, there is correct information about WAL archiving.
	// Verbose mode.
	// When "verboseWAL == true" - WALMin and WALMax are added as metric labels.
	// This creates new different time series on each WAL archiving which maybe is not right way.
	for _, archive := range archiveData {
		if archive.WALMin != "" && archive.WALMax != "" {
			if verboseWAL {
				setUpMetric(
					pgbrWALArchivingMetric,
					"pgbackrest_wal_archive_status",
					1,
					setUpMetricValueFun,
					logger,
					strconv.Itoa(archive.Database.ID),
					getPGVersion(archive.Database.ID, archive.Database.RepoKey, dbData),
					strconv.Itoa(archive.Database.RepoKey),
					stanzaName,
					archive.WALMax,
					archive.WALMin,
				)
			} else {
				setUpMetric(
					pgbrWALArchivingMetric,
					"pgbackrest_wal_archive_status",
					1,
					setUpMetricValueFun,
					logger,
					strconv.Itoa(archive.Database.ID),
					getPGVersion(archive.Database.ID, archive.Database.RepoKey, dbData),
					strconv.Itoa(archive.Database.RepoKey),
					stanzaName,
					"",
					"",
				)
			}
		} else {
			setUpMetric(
				pgbrWALArchivingMetric,
				"pgbackrest_wal_archive_status",
				0,
				setUpMetricValueFun,
				logger,
				strconv.Itoa(archive.Database.ID),
				getPGVersion(archive.Database.ID, archive.Database.RepoKey, dbData),
				strconv.Itoa(archive.Database.RepoKey),
				stanzaName,
				"",
				"",
			)
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

	setUpMetric(
		pgbrExporterInfoMetric,
		"pgbackrest_exporter_info",
		1,
		setUpMetricValueFun,
		logger,
		exporterVer,
	)
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

func setUpMetric(metric *prometheus.GaugeVec, metricName string, value float64, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger, labels ...string) {
	level.Debug(logger).Log(
		"msg", "Set up metric",
		"metric", metricName,
		"value", value,
		"labels", strings.Join(labels, ","),
	)
	err := setUpMetricValueFun(metric, value, labels...)
	if err != nil {
		level.Error(logger).Log(
			"msg", "Metric set up failed",
			"metric", metricName,
			"err", err,
		)
	}
}
