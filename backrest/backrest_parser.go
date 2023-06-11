package backrest

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
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

// Reset all metrics.
func resetMetrics() {
	resetStanzaMetrics()
	resetRepoMetrics()
	resetBackupMetrics()
	resetLastBackupMetrics()
	resetWALMetrics()
}
