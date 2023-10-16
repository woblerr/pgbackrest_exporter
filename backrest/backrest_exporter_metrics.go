package backrest

import (
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	pgbrExporterInfoMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_exporter_info",
		Help: "Information about pgBackRest exporter.",
	},
		[]string{"version"})
	pgbrExporterStatusMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_exporter_status",
		Help: "pgBackRest exporter get data status.",
	},
		[]string{"stanza"})
)

// Set exporter info metrics:
//   - pgbackrest_exporter_info
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

// Set exporter metrics:
//   - pgbackrest_exporter_status
func getExporterStatusMetrics(stanzaName string, getDataStatus bool, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) {
	// If the information is collected for all available stanzas,
	// the value of the label 'stanza' will be 'all-stanzas',
	// otherwise the stanza name will be set.
	if stanzaName == "" {
		stanzaName = "all-stanzas"
	}
	setUpMetric(
		pgbrExporterStatusMetric,
		"pgbackrest_exporter_status",
		convertBoolToFloat64(getDataStatus),
		setUpMetricValueFun,
		logger,
		stanzaName,
	)
}

func resetExporterMetrics() {
	pgbrExporterStatusMetric.Reset()
}
