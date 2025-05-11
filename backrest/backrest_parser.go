package backrest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type setUpMetricValueFunType func(metric *prometheus.GaugeVec, value float64, labels ...string) error

type backupStruct struct {
	backupLabel        string
	backupType         string
	backupTime         time.Time
	backupDuration     float64
	backupDelta        int64
	backupSize         int64
	backupRepoDelta    int64
	backupRepoDeltaMap *int64
	backupRepoSize     *int64
	backupRepoSizeMap  *int64
	backupError        *bool
	backupAnnotation   *annotation
	backupBlockIncr    string
	backupReference    []string
}
type lastBackupsStruct struct {
	full backupStruct
	diff backupStruct
	incr backupStruct
}

var execCommand = exec.Command

const (
	// https://golang.org/pkg/time/#Time.Format
	layout    = "2006-01-02 15:04:05"
	fullLabel = "full"
	diffLabel = "diff"
	incrLabel = "incr"
)

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

func getAllInfoData(config, configIncludePath, stanza, backupType string, logger *slog.Logger) ([]byte, error) {
	var backupLabel string
	return getInfoData(config, configIncludePath, stanza, backupType, backupLabel, logger)
}

func getSpecificBackupInfoData(config, configIncludePath, stanza, backupLabel string, logger *slog.Logger) ([]byte, error) {
	var backupType string
	return getInfoData(config, configIncludePath, stanza, backupType, backupLabel, logger)
}

func getInfoData(config, configIncludePath, stanza, backupType, backupLabel string, logger *slog.Logger) ([]byte, error) {
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
		logger.Error(
			"pgBackRest message",
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
	var stanzas []stanza
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

func compareLastBackups(backups *lastBackupsStruct, currentBackup backup, blockIncr string) {
	currentBackupTime := time.Unix(currentBackup.Timestamp.Stop, 0)
	curentBackupDuration := time.Unix(currentBackup.Timestamp.Stop, 0).Sub(time.Unix(currentBackup.Timestamp.Start, 0)).Seconds()
	currentBackupLabel := currentBackup.Label
	switch currentBackup.Type {
	case "full":
		if currentBackupTime.After(backups.full.backupTime) {
			backups.full.backupTime = currentBackupTime
			backups.full.backupLabel = currentBackupLabel
			backups.full.backupDuration = curentBackupDuration
			backups.full.backupDelta = currentBackup.Info.Delta
			backups.full.backupSize = currentBackup.Info.Size
			backups.full.backupRepoDelta = currentBackup.Info.Repository.Delta
			backups.full.backupRepoDeltaMap = currentBackup.Info.Repository.DeltaMap
			backups.full.backupRepoSize = currentBackup.Info.Repository.Size
			backups.full.backupRepoSizeMap = currentBackup.Info.Repository.SizeMap
			backups.full.backupError = currentBackup.Error
			backups.full.backupAnnotation = currentBackup.Annotation
			backups.full.backupBlockIncr = blockIncr
			backups.full.backupReference = currentBackup.Reference
		}
		if currentBackupTime.After(backups.diff.backupTime) {
			backups.diff.backupTime = currentBackupTime
			backups.diff.backupLabel = currentBackupLabel
			backups.diff.backupDuration = curentBackupDuration
			backups.diff.backupDelta = currentBackup.Info.Delta
			backups.diff.backupSize = currentBackup.Info.Size
			backups.diff.backupRepoDelta = currentBackup.Info.Repository.Delta
			backups.diff.backupRepoDeltaMap = currentBackup.Info.Repository.DeltaMap
			backups.diff.backupRepoSize = currentBackup.Info.Repository.Size
			backups.diff.backupRepoSizeMap = currentBackup.Info.Repository.SizeMap
			backups.diff.backupError = currentBackup.Error
			backups.diff.backupAnnotation = currentBackup.Annotation
			backups.diff.backupBlockIncr = blockIncr
			backups.diff.backupReference = currentBackup.Reference
		}
		if currentBackupTime.After(backups.incr.backupTime) {
			backups.incr.backupTime = currentBackupTime
			backups.incr.backupLabel = currentBackupLabel
			backups.incr.backupDuration = curentBackupDuration
			backups.incr.backupDelta = currentBackup.Info.Delta
			backups.incr.backupSize = currentBackup.Info.Size
			backups.incr.backupRepoDelta = currentBackup.Info.Repository.Delta
			backups.incr.backupRepoDeltaMap = currentBackup.Info.Repository.DeltaMap
			backups.incr.backupRepoSize = currentBackup.Info.Repository.Size
			backups.incr.backupRepoSizeMap = currentBackup.Info.Repository.SizeMap
			backups.incr.backupError = currentBackup.Error
			backups.incr.backupAnnotation = currentBackup.Annotation
			backups.incr.backupBlockIncr = blockIncr
			backups.incr.backupReference = currentBackup.Reference
		}
	case "diff":
		if currentBackupTime.After(backups.diff.backupTime) {
			backups.diff.backupTime = currentBackupTime
			backups.diff.backupLabel = currentBackupLabel
			backups.diff.backupDuration = curentBackupDuration
			backups.diff.backupDelta = currentBackup.Info.Delta
			backups.diff.backupSize = currentBackup.Info.Size
			backups.diff.backupRepoDelta = currentBackup.Info.Repository.Delta
			backups.diff.backupRepoDeltaMap = currentBackup.Info.Repository.DeltaMap
			backups.diff.backupRepoSize = currentBackup.Info.Repository.Size
			backups.diff.backupRepoSizeMap = currentBackup.Info.Repository.SizeMap
			backups.diff.backupError = currentBackup.Error
			backups.diff.backupAnnotation = currentBackup.Annotation
			backups.diff.backupBlockIncr = blockIncr
			backups.diff.backupReference = currentBackup.Reference
		}
		if currentBackupTime.After(backups.incr.backupTime) {
			backups.incr.backupTime = currentBackupTime
			backups.incr.backupLabel = currentBackupLabel
			backups.incr.backupDuration = curentBackupDuration
			backups.incr.backupDelta = currentBackup.Info.Delta
			backups.incr.backupSize = currentBackup.Info.Size
			backups.incr.backupRepoDelta = currentBackup.Info.Repository.Delta
			backups.incr.backupRepoDeltaMap = currentBackup.Info.Repository.DeltaMap
			backups.incr.backupRepoSize = currentBackup.Info.Repository.Size
			backups.incr.backupRepoSizeMap = currentBackup.Info.Repository.SizeMap
			backups.incr.backupError = currentBackup.Error
			backups.incr.backupAnnotation = currentBackup.Annotation
			backups.incr.backupBlockIncr = blockIncr
			backups.incr.backupReference = currentBackup.Reference
		}
	case "incr":
		if currentBackupTime.After(backups.incr.backupTime) {
			backups.incr.backupTime = currentBackupTime
			backups.incr.backupLabel = currentBackupLabel
			backups.incr.backupDuration = curentBackupDuration
			backups.incr.backupDelta = currentBackup.Info.Delta
			backups.incr.backupSize = currentBackup.Info.Size
			backups.incr.backupRepoDelta = currentBackup.Info.Repository.Delta
			backups.incr.backupRepoDeltaMap = currentBackup.Info.Repository.DeltaMap
			backups.incr.backupRepoSize = currentBackup.Info.Repository.Size
			backups.incr.backupRepoSizeMap = currentBackup.Info.Repository.SizeMap
			backups.incr.backupError = currentBackup.Error
			backups.incr.backupAnnotation = currentBackup.Annotation
			backups.incr.backupBlockIncr = blockIncr
			backups.incr.backupReference = currentBackup.Reference
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

func getParsedSpecificBackupInfoData(config, configIncludePath, stanzaName, backupLabel string, logger *slog.Logger) ([]stanza, error) {
	stanzaDataSpecific, err := getSpecificBackupInfoData(config, configIncludePath, stanzaName, backupLabel, logger)
	if err != nil {
		logger.Error(
			"Get data from pgBackRest failed",
			"stanza", stanzaName,
			"backup", backupLabel,
			"err", err)
	}
	parseDataSpecific, err := parseResult(stanzaDataSpecific)
	if err != nil {
		logger.Error(
			"Parse JSON failed",
			"stanza", stanzaName,
			"backup", backupLabel,
			"err", err)
	}
	return parseDataSpecific, err
}

func setUpMetric(metric *prometheus.GaugeVec, metricName string, value float64, setUpMetricValueFun setUpMetricValueFunType, logger *slog.Logger, labels ...string) {
	logger.Debug(
		"Set up metric",
		"metric", metricName,
		"value", value,
		"labels", strings.Join(labels, ","),
	)
	err := setUpMetricValueFun(metric, value, labels...)
	if err != nil {
		logger.Error(
			"Metric set up failed",
			"metric", metricName,
			"err", err,
		)
	}
}

// Reset all metrics.
func resetMetrics() {
	resetStanzaMetrics()
	resetRepoMetrics()
	resetBackupMetrics()
	resetLastBackupMetrics()
	resetWALMetrics()
	resetExporterMetrics()
}

func (backup backup) checkBackupIncremental() string {
	// Block incremental map is used for block level backup.
	// If one value from 'size-map' or 'delta-map' is nil, and other has correct value,
	// it looks like a bug in pgBackRest.
	// See https://github.com/pgbackrest/pgbackrest/blob/3feed389a2199454db68e446851323498b45db20/src/command/info/info.c#L459-L463
	// Relation - backupInfoRepoSizeMap != NULL, where backupInfoRepoSizeMap is related to SizeMap (size-map).
	if backup.Info.Repository.SizeMap != nil && backup.Info.Repository.DeltaMap != nil {
		// The block incremental backup functionality is used.
		return "y"
	}
	return "n"
}

func processSpecificBackupData(config, configIncludePath, stanzaName, backupLabel, backupType, metricName string, metric *prometheus.GaugeVec, setUpMetricValueFun setUpMetricValueFunType, logger *slog.Logger, addLabels ...string) {
	var metricValue float64 = 0
	parseStanzaDataSpecific, err := getParsedSpecificBackupInfoData(config, configIncludePath, stanzaName, backupLabel, logger)
	if err != nil {
		logger.Error(
			"Get data from pgBackRest failed",
			"stanza", stanzaName,
			"backup", backupLabel,
			"err", err,
		)
	}
	// In a normal situation, only one element with one backup should be returned.
	// If more than one element or one backup is returned, there is may be a bug in pgBackRest.
	// If it's not a bug, then this part will need to be refactoring.
	// Use *[]struct() type for backup.DatabaseRef.
	if (len(parseStanzaDataSpecific) != 0 && len(parseStanzaDataSpecific[0].Backup) != 0) &&
		parseStanzaDataSpecific[0].Backup[0].DatabaseRef != nil {
		metricValue = convertDatabaseRefPointerToFloat(parseStanzaDataSpecific[0].Backup[0].DatabaseRef)
	} else {
		logger.Warn(
			"No backup data returned",
			"stanza", stanzaName,
			"backup", backupLabel,
		)
	}
	labels := append([]string{backupType, stanzaName}, addLabels...)
	setUpMetric(
		metric,
		metricName,
		metricValue,
		setUpMetricValueFun,
		logger,
		labels...,
	)
}

// processBackupReferencesCount processes the number of references to another backup (backup reference list).
func processBackupReferencesCount(backupReference []string, metricName string, metric *prometheus.GaugeVec, setUpMetricValueFun setUpMetricValueFunType, logger *slog.Logger, addLabels ...string) {
	refListTotal, err := getBackupReferencesTotal(backupReference)
	if err != nil {
		logger.Error(
			"Failed to get backup references",
			"reference", strings.Join(backupReference, ","),
			"err", err,
		)
	}
	for refType, refCNT := range refListTotal {
		setUpMetric(
			metric,
			metricName,
			float64(refCNT),
			setUpMetricValueFun,
			logger,
			append([]string{refType}, addLabels...)...,
		)
	}
}

// getBackupReferencesTotal counts the number of full, diff and incr backups.
func getBackupReferencesTotal(refList []string) (map[string]int, error) {
	total := map[string]int{
		fullLabel: 0,
		diffLabel: 0,
		incrLabel: 0,
	}
	if len(refList) == 0 {
		return total, nil
	}
	for _, ref := range refList {
		switch {
		case strings.HasSuffix(ref, "F"):
			total[fullLabel]++
		case strings.HasSuffix(ref, "D"):
			total[diffLabel]++
		case strings.HasSuffix(ref, "I"):
			total[incrLabel]++
		default:
			return total, fmt.Errorf("invalid backup name %s", ref)
		}
	}
	return total, nil
}
