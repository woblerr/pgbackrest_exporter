package backrest

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	pgbrVersionInfoMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pgbackrest_version_info",
		Help: "Information about pgBackRest version.",
	}, []string{})
)

// Set version metric:
//   - pgbackrest_version_info
func getBackrestVersionMetrics(setUpMetricValueFun setUpMetricValueFunType, logger *slog.Logger) {
	versionData, err := getVersionData(logger)
	if err != nil {
		logger.Error(
			"Get data from pgBackRest failed",
			"err", err,
		)
	}
	parsedVersionData, err := parseVersionOutput(versionData, logger)
	if err != nil {
		logger.Error(
			"Parse version failed",
			"err", err,
		)
	}
	setUpMetric(
		pgbrVersionInfoMetric,
		"pgbackrest_version_info",
		parsedVersionData,
		setUpMetricValueFun,
		logger,
	)
}

func resetVersionMetrics() {
	pgbrVersionInfoMetric.Reset()
}
