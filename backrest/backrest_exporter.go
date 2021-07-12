package backrest

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	promPort     string
	promEndpoint string
)

// SetPromPortandPath sets HTTP endpoint parameters from command line arguments 'port' and 'endpoint'
func SetPromPortandPath(port, endpoint string) {
	promPort = port
	promEndpoint = endpoint
}

// StartPromEndpoint run HTTP endpoint
func StartPromEndpoint() {
	go func() {
		http.Handle(promEndpoint, promhttp.Handler())
		log.Fatalf("[ERROR] Run HTTP endpoint failed, %v.", http.ListenAndServe(":"+promPort, nil))
	}()
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
	pgbrWALArchivingMetric.Reset()
}

// GetPgBackRestInfo get and parse pgbackrest info and set metrics
func GetPgBackRestInfo(config, configIncludePath, stanza string, verbose bool) error {
	stanzaData, err := getAllInfoData(config, configIncludePath, stanza)
	if err != nil {
		log.Printf("[ERROR] Get data from pgbackrest failed, %v.", err)
		return err
	}
	parseStanzaData, err := parseResult(stanzaData)
	if err != nil {
		log.Printf("[ERROR] Parse JSON failed, %v.", err)
		return err
	}
	if len(parseStanzaData) == 0 {
		log.Printf("[WARN] No backup data returned.")
	}
	for _, singleStanza := range parseStanzaData {
		getMetrics(singleStanza, verbose, setUpMetricValue)
	}
	return nil
}
