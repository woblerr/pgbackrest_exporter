package backrest

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/web"
)

var (
	webFlagsConfig web.FlagConfig
	webEndpoint    string
)

// BackrestExporterConfig contains additional configuration parameters for the pgBackRest exporter.
// Fields correspond to command-line flags with default values applied when empty.
type BackrestExporterConfig struct {
	// Config is the full path to pgBackRest configuration file.
	Config string
	// ConfigIncludePath is the full path to additional pgBackRest configuration files.
	ConfigIncludePath string
	// BackupType is the specific backup type for collecting metrics. One of: [full, incr, diff].
	BackupType string
	// IncludeStanza is the list of specific stanzas for collecting metrics.
	IncludeStanza []string
	// ExcludeStanza is the list of stanzas to exclude from collecting metrics.
	ExcludeStanza []string
	// BackupReferenceCount enables exposing the number of references to other backups.
	BackupReferenceCount bool
	// BackupDBCount enables exposing the number of databases in backups.
	BackupDBCount bool
	// BackupDBCountLatest enables exposing the number of databases in the latest backups.
	BackupDBCountLatest bool
	// VerboseWAL enables additional labels for WAL metrics.
	VerboseWAL bool
	// ResetMetricsAfter determines when to reset metrics.
	// If true, metrics are reset after receiving data from pgBackRest.
	// If false, metrics are reset before the loop (used when specific stanzas are included).
	ResetMetricsAfter bool
	// BackupDBCountParallelProcesses is the number of parallel processes for collecting database information.
	BackupDBCountParallelProcesses int
}

// LogBackrestExporterConfig logs BackrestExporterConfig parameters.
func LogBackrestExporterConfig(cfg BackrestExporterConfig, logger *slog.Logger) {
	// Log pgBackRest configuration parameters.
	if cfg.Config != "" {
		logger.Info(
			"Custom pgBackRest configuration file",
			"file", cfg.Config)
	}
	if cfg.ConfigIncludePath != "" {
		logger.Info(
			"Custom path to additional pgBackRest configuration files",
			"path", cfg.ConfigIncludePath)
	}
	if strings.Join(cfg.IncludeStanza, "") != "" {
		for _, stanza := range cfg.IncludeStanza {
			logger.Info(
				"Collecting metrics for specific stanza",
				"stanza", stanza)
		}
	}
	if strings.Join(cfg.ExcludeStanza, "") != "" {
		for _, stanza := range cfg.ExcludeStanza {
			logger.Info(
				"Exclude collecting metrics for specific stanza",
				"stanza", stanza)
		}
	}
	if cfg.BackupType != "" {
		logger.Info(
			"Collecting metrics for specific backup type",
			"type", cfg.BackupType)
	}
	if cfg.BackupReferenceCount {
		logger.Info(
			"Exposing the number of references to other backups (backup reference list)",
			"reference-count", cfg.BackupReferenceCount)
	}
	if cfg.BackupDBCount {
		logger.Info(
			"Exposing the number of databases in backups",
			"database-count", cfg.BackupDBCount,
			"database-parallel-processes", cfg.BackupDBCountParallelProcesses)
	}
	if cfg.BackupDBCountLatest {
		logger.Info(
			"Exposing the number of databases in the latest backups",
			"database-count-latest", cfg.BackupDBCountLatest)
	}
	if cfg.VerboseWAL {
		logger.Info(
			"Enabling additional labels for WAL metrics",
			"verbose-wal", cfg.VerboseWAL)
	}
}

// SetPromPortAndPath sets HTTP endpoint parameters
// from command line arguments:
// 'web.telemetry-path',
// 'web.listen-address',
// 'web.config.file',
// 'web.systemd-socket' (Linux only)
func SetPromPortAndPath(flagsConfig web.FlagConfig, endpoint string) {
	webFlagsConfig = flagsConfig
	webEndpoint = endpoint
}

// StartPromEndpoint run HTTP endpoint
func StartPromEndpoint(version string, logger *slog.Logger) {
	go func(logger *slog.Logger) {
		if webEndpoint == "" {
			logger.Error("Metric endpoint is empty", "endpoint", webEndpoint)
		}
		http.Handle(webEndpoint, promhttp.Handler())
		if webEndpoint != "/" {
			landingConfig := web.LandingConfig{
				Name:        "pgBackRest exporter",
				Description: "Prometheus exporter for pgBackRest",
				HeaderColor: "#476b6b",
				Version:     version,
				Profiling:   "false",
				Links: []web.LandingLinks{
					{
						Address: webEndpoint,
						Text:    "Metrics",
					},
				},
			}
			landingPage, err := web.NewLandingPage(landingConfig)
			if err != nil {
				logger.Error("Error creating landing page", "err", err)
				os.Exit(1)
			}
			http.Handle("/", landingPage)
		}
		server := &http.Server{
			ReadHeaderTimeout: 5 * time.Second,
		}
		if err := web.ListenAndServe(server, &webFlagsConfig, logger); err != nil {
			logger.Error("Run web endpoint failed", "err", err)
			os.Exit(1)
		}
	}(logger)
}

// GetPgBackRestInfo get and parse pgBackRest info and set metrics
func GetPgBackRestInfo(cfg BackrestExporterConfig, logger *slog.Logger) {
	// To calculate the time elapsed since the last completed full, differential or incremental backup.
	// For all stanzas values are calculated relative to one value.
	currentUnixTime := time.Now().Unix()
	// If specific stanzas are specified for collecting metrics,
	// then we reset all metrics before the loop.
	// Otherwise, it makes sense to reset the metrics after receiving data from pgBackRest,
	// because this operation can be long.
	if !cfg.ResetMetricsAfter {
		resetMetrics()
	}
	// Determine if exclude flag is specified (non-empty list).
	excludeSpecified := strings.Join(cfg.ExcludeStanza, "") != ""
	// Loop over each stanza.
	// If stanza not set - perform a single loop step to get metrics for all stanzas.
	for _, stanza := range cfg.IncludeStanza {
		// Flag to check if pgBackRest get info for this stanza.
		// By default, it's set to true.
		// If we get an error from pgBackRest when getting info for stanza, flag will be set to false.
		getDataSuccessStatus := true
		// Check that stanza from the include list is not in the exclude list.
		// If stanza not set - checking for entry into the exclude list will be performed later.
		if !stanzaInExclude(stanza, cfg.ExcludeStanza) {
			stanzaData, err := getAllInfoData(cfg.Config, cfg.ConfigIncludePath, stanza, cfg.BackupType, logger)
			if err != nil {
				getDataSuccessStatus = false
				logger.Error("Get data from pgBackRest failed", "err", err)
			}
			parseStanzaData, err := parseResult(stanzaData)
			if err != nil {
				getDataSuccessStatus = false
				logger.Error("Parse JSON failed", "err", err)
			}
			if len(parseStanzaData) == 0 {
				logger.Warn("No backup data returned")
			}
			// When no specific stanzas set for collecting we can reset the metrics as late as possible.
			if cfg.ResetMetricsAfter {
				resetMetrics()
			}
			getExporterStatusMetrics(stanza, getDataSuccessStatus, excludeSpecified, setUpMetricValue, logger)
			for _, singleStanza := range parseStanzaData {
				// If stanza is in the exclude list, skip it.
				if stanzaInExclude(singleStanza.Name, cfg.ExcludeStanza) {
					continue
				}
				getStanzaMetrics(singleStanza.Name, singleStanza.Status, setUpMetricValue, logger)
				getRepoMetrics(singleStanza.Name, singleStanza.Repo, setUpMetricValue, logger)
				getWALMetrics(singleStanza.Name, singleStanza.Archive, singleStanza.DB, cfg.VerboseWAL, setUpMetricValue, logger)
				// Last backups for current stanza
				lastBackups := getBackupMetrics(singleStanza.Name, cfg.BackupReferenceCount, singleStanza.Backup, singleStanza.DB, setUpMetricValue, logger)
				// If full backup exists, the values of metrics for differential and
				// incremental backups also will be set.
				// If not - metrics won't be set.
				if !lastBackups.full.backupTime.IsZero() {
					getBackupLastMetrics(singleStanza.Name, lastBackups, currentUnixTime, setUpMetricValue, logger)
				}
				// If the calculation of the number of databases in backups is enabled.
				// Information about number of databases in specific backup has appeared since pgBackRest v2.41.
				// In versions < v2.41 this is missing and the metric will be set to 0.
				if cfg.BackupDBCount {
					getBackupDBCountMetrics(cfg.BackupDBCountParallelProcesses, cfg.Config, cfg.ConfigIncludePath, singleStanza.Name, singleStanza.Backup, setUpMetricValue, logger)
				}
				// If the calculation of the number of databases in latest backups is enabled.
				// Information about number of databases in specific backup has appeared since pgBackRest v2.41.
				// In versions < v2.41 this is missing and the metric will be set to 0.
				if cfg.BackupDBCountLatest && !lastBackups.full.backupTime.IsZero() {
					getBackupLastDBCountMetrics(cfg.Config, cfg.ConfigIncludePath, singleStanza.Name, lastBackups, setUpMetricValue, logger)
				}
			}
		} else {
			// When stanza is specified in both include and exclude lists, a warning is displayed in the log
			// and data for this stanza is not collected.
			// It is necessary to set zero metric value for this stanza.
			getDataSuccessStatus = false
			getExporterStatusMetrics(stanza, getDataSuccessStatus, excludeSpecified, setUpMetricValue, logger)
			logger.Warn("Stanza is specified in include and exclude lists", "stanza", stanza)
		}
	}
}

// GetPgBackrestVersionInfo get and parse pgBackRest version info and set metrics
func GetPgBackrestVersionInfo(logger *slog.Logger) {
	resetVersionMetrics()
	getBackrestVersionMetrics(setUpMetricValue, logger)
}
