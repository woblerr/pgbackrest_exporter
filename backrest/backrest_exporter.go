package backrest

import (
	"net/http"
	"os"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/web"
)

var (
	webFlagsConfig web.FlagConfig
	webEndpoint    string
	// When reset metrics.
	// Before receiving information from pgBackRest (false) or after (true).
	MetricResetFlag bool = true
)

// SetPromPortAndPath sets HTTP endpoint parameters
// from command line arguments:
// 'web.endpoint',
// 'web.listen-address',
// 'web.config.file',
// 'web.systemd-socket' (Linux only)
func SetPromPortAndPath(flagsConfig web.FlagConfig, endpoint string) {
	webFlagsConfig = flagsConfig
	webEndpoint = endpoint
}

// StartPromEndpoint run HTTP endpoint
func StartPromEndpoint(logger log.Logger) {
	go func(logger log.Logger) {
		http.Handle(webEndpoint, promhttp.Handler())
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<html>
			<head><title>pgBackRest exporter</title></head>
			<body>
			<h1>pgBackRest exporter</h1>
			<p><a href='` + webEndpoint + `'>Metrics</a></p>
			</body>
			</html>`))
		})
		server := &http.Server{
			ReadHeaderTimeout: 5 * time.Second,
		}
		if err := web.ListenAndServe(server, &webFlagsConfig, logger); err != nil {
			level.Error(logger).Log("msg", "Run web endpoint failed", "err", err)
			os.Exit(1)
		}
	}(logger)
}

// GetPgBackRestInfo get and parse pgBackRest info and set metrics
func GetPgBackRestInfo(config, configIncludePath, backupType string, stanzas, stanzasExclude []string, backupDBCount, backupDBCountLatest, verboseWAL bool, backupDBCountParallelProcesses int, logger log.Logger) {
	// To calculate the time elapsed since the last completed full, differential or incremental backup.
	// For all stanzas values are calculated relative to one value.
	currentUnixTime := time.Now().Unix()
	// If specific stanzas are specified for collecting metrics,
	// then we reset all metrics before the loop.
	// Otherwise, it makes sense to reset the metrics after receiving data from pgBackRest,
	// because this operation can be long.
	if !MetricResetFlag {
		resetMetrics()
	}
	// Loop over each stanza.
	// If stanza not set - perform a single loop step to get metrics for all stanzas.
	for _, stanza := range stanzas {
		// Flag to check if pgBackRest get info for this stanza.
		// By default, it's set to true.
		// If we get an error from pgBackRest when getting info for stanza, flag will be set to false.
		getDataSuccessStatus := true
		// Check that stanza from the include list is not in the exclude list.
		// If stanza not set - checking for entry into the exclude list will be performed later.
		if stanzaNotInExclude(stanza, stanzasExclude) {
			stanzaData, err := getAllInfoData(config, configIncludePath, stanza, backupType, logger)
			if err != nil {
				getDataSuccessStatus = false
				level.Error(logger).Log("msg", "Get data from pgBackRest failed", "err", err)
			}
			parseStanzaData, err := parseResult(stanzaData)
			if err != nil {
				getDataSuccessStatus = false
				level.Error(logger).Log("msg", "Parse JSON failed", "err", err)
			}
			if len(parseStanzaData) == 0 {
				level.Warn(logger).Log("msg", "No backup data returned")
			}
			// When no specific stanzas set for collecting we can reset the metrics as late as possible.
			if MetricResetFlag {
				resetMetrics()
			}
			getExporterStatusMetrics(stanza, getDataSuccessStatus, setUpMetricValue, logger)
			for _, singleStanza := range parseStanzaData {
				// If stanza is in the exclude list, skip it.
				if stanzaNotInExclude(singleStanza.Name, stanzasExclude) {
					getStanzaMetrics(singleStanza.Name, singleStanza.Status, setUpMetricValue, logger)
					getRepoMetrics(singleStanza.Name, singleStanza.Repo, setUpMetricValue, logger)
					getWALMetrics(singleStanza.Name, singleStanza.Archive, singleStanza.DB, verboseWAL, setUpMetricValue, logger)
					// Last backups for current stanza
					lastBackups := getBackupMetrics(singleStanza.Name, singleStanza.Backup, singleStanza.DB, setUpMetricValue, logger)
					// If full backup exists, the values of metrics for differential and
					// incremental backups also will be set.
					// If not - metrics won't be set.
					if !lastBackups.full.backupTime.IsZero() {
						getBackupLastMetrics(singleStanza.Name, lastBackups, currentUnixTime, setUpMetricValue, logger)
					}
					// If the calculation of the number of databases in backups is enabled.
					// Information about number of databases in specific backup has appeared since pgBackRest v2.41.
					// In versions < v2.41 this is missing and the metric will be set to 0.
					if backupDBCount {
						getBackupDBCountMetrics(backupDBCountParallelProcesses, config, configIncludePath, singleStanza.Name, singleStanza.Backup, setUpMetricValue, logger)
					}
					// If the calculation of the number of databases in latest backups is enabled.
					// Information about number of databases in specific backup has appeared since pgBackRest v2.41.
					// In versions < v2.41 this is missing and the metric will be set to 0.
					if backupDBCountLatest && !lastBackups.full.backupTime.IsZero() {
						getBackupLastDBCountMetrics(config, configIncludePath, singleStanza.Name, lastBackups, setUpMetricValue, logger)
					}
				}
			}
		} else {
			// When stanza is specified in both include and exclude lists, a warning is displayed in the log
			// and data for this stanza is not collected.
			// It is necessary to set zero metric value for this stanza.
			getDataSuccessStatus = false
			getExporterStatusMetrics(stanza, getDataSuccessStatus, setUpMetricValue, logger)
			level.Warn(logger).Log("msg", "Stanza is specified in include and exclude lists", "stanza", stanza)
		}
	}
}

// GetExporterInfo set exporter info metric
func GetExporterInfo(exporterVersion string, logger log.Logger) {
	getExporterMetrics(exporterVersion, setUpMetricValue, logger)
}
