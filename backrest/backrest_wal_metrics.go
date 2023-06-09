package backrest

import (
	"strconv"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var pgbrWALArchivingMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "pgbackrest_wal_archive_status",
	Help: "Current WAL archive status.",
},
	[]string{
		"database_id",
		"pg_version",
		"repo_key",
		"stanza",
		"wal_max",
		"wal_min"})

// Set backup metrics:
//   - pgbackrest_wal_archive_status
func getWALMetrics(stanzaName string, archiveData []archive, dbData []db, verboseWAL bool, setUpMetricValueFun setUpMetricValueFunType, logger log.Logger) {
	// WAL archive info.
	// 0 - any one of WALMin and WALMax have empty value, there is no correct information about WAL archiving.
	// 1 - both WALMin and WALMax have no empty values, there is correct information about WAL archiving.
	// Verbose mode.
	// When "verboseWAL == true" - WALMin and WALMax are added as metric labels.
	// This creates new different time series on each WAL archiving which maybe is not right way.
	for _, archive := range archiveData {
		if archive.WALMin != "" && archive.WALMax != "" {
			if verboseWAL {
				setUpMetric(
					pgbrWALArchivingMetric,
					"pgbackrest_wal_archive_status",
					1,
					setUpMetricValueFun,
					logger,
					strconv.Itoa(archive.Database.ID),
					getPGVersion(archive.Database.ID, archive.Database.RepoKey, dbData),
					strconv.Itoa(archive.Database.RepoKey),
					stanzaName,
					archive.WALMax,
					archive.WALMin,
				)
			} else {
				setUpMetric(
					pgbrWALArchivingMetric,
					"pgbackrest_wal_archive_status",
					1,
					setUpMetricValueFun,
					logger,
					strconv.Itoa(archive.Database.ID),
					getPGVersion(archive.Database.ID, archive.Database.RepoKey, dbData),
					strconv.Itoa(archive.Database.RepoKey),
					stanzaName,
					"",
					"",
				)
			}
		} else {
			setUpMetric(
				pgbrWALArchivingMetric,
				"pgbackrest_wal_archive_status",
				0,
				setUpMetricValueFun,
				logger,
				strconv.Itoa(archive.Database.ID),
				getPGVersion(archive.Database.ID, archive.Database.RepoKey, dbData),
				strconv.Itoa(archive.Database.RepoKey),
				stanzaName,
				"",
				"",
			)
		}
	}
}

func resetWALMetrics() {
	pgbrWALArchivingMetric.Reset()
}
