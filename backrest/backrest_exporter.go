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
	promPort          string
	promEndpoint      string
	promTLSConfigPath string
)

// SetPromPortAndPath sets HTTP endpoint parameters
// from command line arguments 'prom.port', 'prom.endpoint' and 'prom.web-config'
func SetPromPortAndPath(port, endpoint, tlsConfigPath string) {
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
	pgbrStanzaBackupLastDatabasesMetric.Reset()
	pgbrWALArchivingMetric.Reset()
}

// GetPgBackRestInfo get and parse pgBackRest info and set metrics
func GetPgBackRestInfo(config, configIncludePath, backupType string, stanzas []string, stanzasExclude []string, backupDBCountLatest, verboseWAL bool, logger log.Logger) {
	// To calculate the time elapsed since the last completed full, differential or incremental backup.
	// For all stanzas values are calculated relative to one value.
	currentUnixTime := time.Now().Unix()
	// Loop over each stanza.
	// If stanza not set - perform a single loop step to get metrics for all stanzas.
	for _, stanza := range stanzas {
		// Check that stanza from the include list is not in the exclude list.
		// If stanza not set - checking for entry into the exclude list will be performed later.
		if stanzaNotInExclude(stanza, stanzasExclude) {
			stanzaData, err := getAllInfoData(config, configIncludePath, stanza, backupType, logger)
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
					getMetrics(config, configIncludePath, singleStanza, backupDBCountLatest, verboseWAL, currentUnixTime, setUpMetricValue, logger)
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
