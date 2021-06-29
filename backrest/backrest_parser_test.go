package backrest

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

func TestGetMetrics(t *testing.T) {
	type args struct {
		data     stanza
		verbose  bool
		testText string
	}
	templateMetrics := `# HELP pgbackrest_exporter_backup_database_size Full uncompressed size of the database.
# TYPE pgbackrest_exporter_backup_database_size gauge
pgbackrest_exporter_backup_database_size{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_exporter_backup_duration Backup duration in seconds.
# TYPE pgbackrest_exporter_backup_duration gauge
pgbackrest_exporter_backup_duration{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo",start_time="2021-06-07 12:24:23",stop_time="2021-06-07 12:24:26"} 3
# HELP pgbackrest_exporter_backup_info Backup info.
# TYPE pgbackrest_exporter_backup_info gauge
pgbackrest_exporter_backup_info{backrest_ver="2.34",backup_name="20210607-092423F",backup_type="full",database_id="1",pg_version="13",prior="",repo_key="1",stanza="demo",wal_archive_max="000000010000000000000002",wal_archive_min="000000010000000000000002"} 1
# HELP pgbackrest_exporter_backup_repo_backup_set_size Full compressed files size to restore the database from backup.
# TYPE pgbackrest_exporter_backup_repo_backup_set_size gauge
pgbackrest_exporter_backup_repo_backup_set_size{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_exporter_backup_repo_backup_size Compressed files size in backup.
# TYPE pgbackrest_exporter_backup_repo_backup_size gauge
pgbackrest_exporter_backup_repo_backup_size{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_exporter_backup_size Amount of data in the database to actually backup.
# TYPE pgbackrest_exporter_backup_size gauge
pgbackrest_exporter_backup_size{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_exporter_repo_status Current repository status.
# TYPE pgbackrest_exporter_repo_status gauge
pgbackrest_exporter_repo_status{cipher="none",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_exporter_stanza_status Current stanza status.
# TYPE pgbackrest_exporter_stanza_status gauge
pgbackrest_exporter_stanza_status{stanza="demo"} 0
# HELP pgbackrest_exporter_wal_archive_status Current WAL archive status.
# TYPE pgbackrest_exporter_wal_archive_status gauge
`
	templateStanza := func(WALMax, WALMin string) stanza {
		return stanza{
			[]archive{
				{databaseID{1, 1}, "13-1", WALMax, WALMin},
			},
			[]backup{
				{struct {
					StartWAL string "json:\"start\""
					StopWAL  string "json:\"stop\""
				}{"000000010000000000000002", "000000010000000000000002"},
					struct {
						Format  int    "json:\"format\""
						Version string "json:\"version\""
					}{5, "2.34"},
					databaseID{1, 1},
					backupInfo{
						24316343,
						struct {
							Delta int64 "json:\"delta\""
							Size  int64 "json:\"size\""
						}{2969514, 2969514},
						24316343,
					},
					"20210607-092423F",
					"",
					[]string{""},
					struct {
						Start int64 "json:\"start\""
						Stop  int64 "json:\"stop\""
					}{1623057863, 1623057866},
					"full",
				},
			},
			"none",
			[]db{
				{1, 1, 6970977677138971135, "13"},
			},
			"demo",
			[]repo{
				{"none",
					1,
					struct {
						Code    int    "json:\"code\""
						Message string "json:\"message\""
					}{0, "ok"},
				},
			},
			status{
				0,
				struct {
					Backup struct {
						Held bool "json:\"held\""
					} "json:\"backup\""
				}{
					struct {
						Held bool "json:\"held\""
					}{false},
				},
				"ok",
			},
		}
	}
	tests := []struct {
		name string
		args args
	}{
		{"getMetricsVerboseFalse",
			args{
				templateStanza("000000010000000000000004", "000000010000000000000001"),
				false,
				templateMetrics +
					`pgbackrest_exporter_wal_archive_status{database_id="1",pg_version="13",repo_key="1",stanza="demo",wal_archive_max="",wal_archive_min=""} 1` +
					"\n",
			},
		},
		{"getMetricsVerboseTrue",
			args{
				templateStanza("000000010000000000000004", "000000010000000000000001"),
				true,
				templateMetrics +
					`pgbackrest_exporter_wal_archive_status{database_id="1",pg_version="13",repo_key="1",stanza="demo",wal_archive_max="000000010000000000000004",wal_archive_min="000000010000000000000001"} 1` +
					"\n",
			},
		},
		{"getMetricsWithoutWal",
			args{
				templateStanza("", "000000010000000000000001"),
				false,
				templateMetrics +
					`pgbackrest_exporter_wal_archive_status{database_id="1",pg_version="13",repo_key="1",stanza="demo",wal_archive_max="",wal_archive_min=""} 0` +
					"\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getMetrics(tt.args.data, tt.args.verbose)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaStatusMetric,
				pgbrRepoStatusMetric,
				pgbrStanzaBackupInfoMetric,
				pgbrStanzaBackupDurationMetric,
				pgbrStanzaBackupSizeMetric,
				pgbrStanzaBackupDatabaseSizeMetric,
				pgbrStanzaRepoBackupSetSizeMetric,
				pgbrStanzaRepoBackupSizeMetric,
				pgbrWALArchivingMetric,
			)
			metricFamily, err := reg.Gather()
			if err != nil {
				fmt.Println(err)
			}
			out := &bytes.Buffer{}
			for _, mf := range metricFamily {
				if _, err := expfmt.MetricFamilyToText(out, mf); err != nil {
					panic(err)
				}
			}
			if tt.args.testText != out.String() {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", tt.args.testText, out.String())
			}
			resetMetrics()
		})
	}
}

func TestGetPGVersion(t *testing.T) {
	type args struct {
		id      int
		repoKey int
		dbList  []db
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"getPGVersionSame",
			args{1, 1, []db{{1, 1, 6970977677138971135, "13"}}},
			"13",
		},
		{"getPGVersionDiff",
			args{1, 5, []db{{1, 1, 6970977677138971135, "13"}}},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPGVersion(tt.args.id, tt.args.repoKey, tt.args.dbList); got != tt.want {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestSetUpMetricValue(t *testing.T) {
	type args struct {
		metric *prometheus.GaugeVec
		value  float64
		labels []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"setUpMetricValueError",
			args{pgbrStanzaStatusMetric, 0, []string{"demo", "bad"}},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := setUpMetricValue(tt.args.metric, tt.args.value, tt.args.labels...); (err != nil) != tt.wantErr {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", err, tt.wantErr)
			}
		})
	}
}
