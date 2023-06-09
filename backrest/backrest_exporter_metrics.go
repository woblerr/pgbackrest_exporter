package backrest

import (
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var pgbrExporterInfoMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "pgbackrest_exporter_info",
	Help: "Information about pgBackRest exporter.",
},
	[]string{"version"})

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
