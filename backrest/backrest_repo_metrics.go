package backrest

import (
	"strconv"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var pgbrRepoStatusMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "pgbackrest_repo_status",
	Help: "Current repository status.",
},
	[]string{
		"cipher",
		"repo_key",
		"stanza",
	})

// Set repo metrics:
//   - pgbackrest_repo_status
func getRepoMetrics(stanzaName string, repoData []repo, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) {
	// Repo status.
	// The same statuses as for stanza.
	for _, repo := range repoData {
		setUpMetric(
			pgbrRepoStatusMetric,
			"pgbackrest_repo_status",
			float64(repo.Status.Code),
			setUpMetricValueFun,
			logger,
			repo.Cipher,
			strconv.Itoa(repo.Key),
			stanzaName,
		)
	}
}

func resetRepoMetrics() {
	pgbrRepoStatusMetric.Reset()
}
