package backrest

import (
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var pgbrStanzaStatusMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "pgbackrest_stanza_status",
	Help: "Current stanza status.",
},
	[]string{"stanza"})

// Set stanza metrics:
//   - pgbackrest_stanza_status
func getStanzaMetrics(stanzaName string, stanzaStatusCode int, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) {
	//https://github.com/pgbackrest/pgbackrest/blob/03021c6a17f1374e84ef42614fa1dd2a6be4b64d/src/command/info/info.c#L78-L94
	// Stanza statuses:
	//  0: "ok",
	//  1: "missing stanza path",
	//  2: "no valid backups",
	//  3: "missing stanza data",
	//  4: "different across repos",
	//  5: "database mismatch across repos",
	//  6: "requested backup not found",
	//  99: "other".
	setUpMetric(
		pgbrStanzaStatusMetric,
		"pgbackrest_stanza_status",
		float64(stanzaStatusCode),
		setUpMetricValueFun,
		logger,
		stanzaName,
	)
}

func resetStanzaMetrics() {
	pgbrStanzaStatusMetric.Reset()
}
