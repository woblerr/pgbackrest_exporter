package backrest

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	pgbrExporterStatusMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_exporter_status",
		Help: "pgBackRest exporter get data status.",
	},
		[]string{"stanza"})
)

// Set exporter metrics:
//   - pgbackrest_exporter_status
func getExporterStatusMetrics(stanzaName string, getDataStatus bool, setUpMetricValueFun setUpMetricValueFunType, logger *slog.Logger) {
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
