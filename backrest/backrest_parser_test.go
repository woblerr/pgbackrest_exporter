package backrest

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

func TestGetMetrics(t *testing.T) {
	type args struct {
		data                stanza
		verbose             bool
		testText            string
		setUpMetricValueFun setUpMetricValueFunType
	}
	templateMetrics := `# HELP pgbackrest_exporter_backup_database_backup_size Amount of data in the database to actually backup.
# TYPE pgbackrest_exporter_backup_database_backup_size gauge
pgbackrest_exporter_backup_database_backup_size{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_exporter_backup_database_size Full uncompressed size of the database.
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
# HELP pgbackrest_exporter_repo_status Current repository status.
# TYPE pgbackrest_exporter_repo_status gauge
pgbackrest_exporter_repo_status{cipher="none",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_exporter_stanza_status Current stanza status.
# TYPE pgbackrest_exporter_stanza_status gauge
pgbackrest_exporter_stanza_status{stanza="demo"} 0
# HELP pgbackrest_exporter_wal_archive_status Current WAL archive status.
# TYPE pgbackrest_exporter_wal_archive_status gauge
`
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
				setUpMetricValue,
			},
		},
		{"getMetricsVerboseTrue",
			args{
				templateStanza("000000010000000000000004", "000000010000000000000001"),
				true,
				templateMetrics +
					`pgbackrest_exporter_wal_archive_status{database_id="1",pg_version="13",repo_key="1",stanza="demo",wal_archive_max="000000010000000000000004",wal_archive_min="000000010000000000000001"} 1` +
					"\n",
				setUpMetricValue,
			},
		},
		{"getMetricsWithoutWal",
			args{
				templateStanza("", "000000010000000000000001"),
				false,
				templateMetrics +
					`pgbackrest_exporter_wal_archive_status{database_id="1",pg_version="13",repo_key="1",stanza="demo",wal_archive_max="",wal_archive_min=""} 0` +
					"\n",
				setUpMetricValue,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			getMetrics(tt.args.data, tt.args.verbose, tt.args.setUpMetricValueFun)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaStatusMetric,
				pgbrRepoStatusMetric,
				pgbrStanzaBackupInfoMetric,
				pgbrStanzaBackupDurationMetric,
				pgbrStanzaBackupDatabaseSizeMetric,
				pgbrStanzaBackupDatabaseBackupSizeMetric,
				pgbrStanzaBackupRepoBackupSetSizeMetric,
				pgbrStanzaBackupRepoBackupSizeMetric,
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
		})
	}
}

func TestGetMetricsRepoAbsent(t *testing.T) {
	type args struct {
		data                stanza
		verbose             bool
		testText            string
		setUpMetricValueFun setUpMetricValueFunType
	}
	templateMetrics := `# HELP pgbackrest_exporter_backup_database_backup_size Amount of data in the database to actually backup.
# TYPE pgbackrest_exporter_backup_database_backup_size gauge
pgbackrest_exporter_backup_database_backup_size{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_exporter_backup_database_size Full uncompressed size of the database.
# TYPE pgbackrest_exporter_backup_database_size gauge
pgbackrest_exporter_backup_database_size{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_exporter_backup_duration Backup duration in seconds.
# TYPE pgbackrest_exporter_backup_duration gauge
pgbackrest_exporter_backup_duration{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo",start_time="2021-06-07 12:24:23",stop_time="2021-06-07 12:24:26"} 3
# HELP pgbackrest_exporter_backup_info Backup info.
# TYPE pgbackrest_exporter_backup_info gauge
pgbackrest_exporter_backup_info{backrest_ver="2.34",backup_name="20210607-092423F",backup_type="full",database_id="1",pg_version="13",prior="",repo_key="0",stanza="demo",wal_archive_max="000000010000000000000002",wal_archive_min="000000010000000000000002"} 1
# HELP pgbackrest_exporter_backup_repo_backup_set_size Full compressed files size to restore the database from backup.
# TYPE pgbackrest_exporter_backup_repo_backup_set_size gauge
pgbackrest_exporter_backup_repo_backup_set_size{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo"} 2.969514e+06
# HELP pgbackrest_exporter_backup_repo_backup_size Compressed files size in backup.
# TYPE pgbackrest_exporter_backup_repo_backup_size gauge
pgbackrest_exporter_backup_repo_backup_size{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo"} 2.969514e+06
# HELP pgbackrest_exporter_stanza_status Current stanza status.
# TYPE pgbackrest_exporter_stanza_status gauge
pgbackrest_exporter_stanza_status{stanza="demo"} 0
# HELP pgbackrest_exporter_wal_archive_status Current WAL archive status.
# TYPE pgbackrest_exporter_wal_archive_status gauge
`
	tests := []struct {
		name string
		args args
	}{
		{"getMetricsVerboseFalse",
			args{
				templateStanzaRepoAbsent("000000010000000000000004", "000000010000000000000001"),
				false,
				templateMetrics +
					`pgbackrest_exporter_wal_archive_status{database_id="1",pg_version="13",repo_key="0",stanza="demo",wal_archive_max="",wal_archive_min=""} 1` +
					"\n",
				setUpMetricValue,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			getMetrics(tt.args.data, tt.args.verbose, tt.args.setUpMetricValueFun)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaStatusMetric,
				pgbrRepoStatusMetric,
				pgbrStanzaBackupInfoMetric,
				pgbrStanzaBackupDurationMetric,
				pgbrStanzaBackupDatabaseSizeMetric,
				pgbrStanzaBackupDatabaseBackupSizeMetric,
				pgbrStanzaBackupRepoBackupSetSizeMetric,
				pgbrStanzaBackupRepoBackupSizeMetric,
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
		})
	}
}

func TestGetMetricsErrors(t *testing.T) {
	type args struct {
		data                stanza
		verbose             bool
		setUpMetricValueFun setUpMetricValueFunType
		errorsCount         int
	}
	tests := []struct {
		name string
		args args
	}{
		{"getMetricsVerboseFalseLogError",
			args{
				templateStanza("000000010000000000000004", "000000010000000000000001"),
				false,
				fakeSetUpMetricValue,
				9,
			},
		},
		{"getMetricsVerboseTrueLogError",
			args{
				templateStanza("000000010000000000000004", "000000010000000000000001"),
				true,
				fakeSetUpMetricValue,
				9,
			},
		},
		{"getMetricsWithoutWalLogError",
			args{
				templateStanza("", "000000010000000000000001"),
				false,
				fakeSetUpMetricValue,
				9,
			},
		},
		{"getMetricsVerboseFalseLogErrorRepoAbsent",
			args{
				templateStanzaRepoAbsent("000000010000000000000004", "000000010000000000000001"),
				false,
				fakeSetUpMetricValue,
				8,
			},
		},
		{"getMetricsVerboseTrueLogErrorRepoAbsent",
			args{
				templateStanzaRepoAbsent("000000010000000000000004", "000000010000000000000001"),
				true,
				fakeSetUpMetricValue,
				8,
			},
		},
		{"getMetricsWithoutWalLogErrorRepoAbsent",
			args{
				templateStanzaRepoAbsent("", "000000010000000000000001"),
				false,
				fakeSetUpMetricValue,
				8,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			log.SetOutput(out)
			getMetrics(tt.args.data, tt.args.verbose, tt.args.setUpMetricValueFun)
			errorsOutputCount := strings.Count(out.String(), "[ERROR]")
			if tt.args.errorsCount != errorsOutputCount {
				t.Errorf("\nVariables do not match:\n%d\nwant:\n%d", tt.args.errorsCount, errorsOutputCount)
			}
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

func TestReturnDefaultExecArgs(t *testing.T) {
	testArgs := []string{"info", "--output", "json"}
	defaultArgs := returnDefaultExecArgs()
	if !reflect.DeepEqual(testArgs, defaultArgs) {
		t.Errorf("\nVariables do not match: %s,\nwant: %s", testArgs, defaultArgs)
	}
}

func TestReturnConfigExecArgs(t *testing.T) {
	type args struct {
		config            string
		configIncludePath string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"returnConfigExecArgsEmpty",
			args{"", ""},
			[]string{},
		},
		{"returnConfigExecArgsNotEmptyConfig",
			args{"/tmp/pgbackrest.conf", ""},
			[]string{"--config", "/tmp/pgbackrest.conf"},
		},
		{"returnConfigExecArgsNotEmptyConfigInckudePath",
			args{"", "/tmp/pgbackrest/conf.d"},
			[]string{"--config-include-path", "/tmp/pgbackrest/conf.d"},
		},
		{"returnConfigExecArgsNotEmptyConfigAndConfigInckudePath",
			args{"/tmp/pgbackrest.conf", "/tmp/pgbackrest/conf.d"},
			[]string{"--config", "/tmp/pgbackrest.conf", "--config-include-path", "/tmp/pgbackrest/conf.d"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := returnConfigExecArgs(tt.args.config, tt.args.configIncludePath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestReturnConfigStanzaArgs(t *testing.T) {
	type args struct {
		stanza string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"returnStanzaExecArgsEmpty",
			args{""},
			[]string{},
		},
		{"returnStanzaExecArgsNotEmpty",
			args{"demo"},
			[]string{"--stanza", "demo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := returnConfigStanzaArgs(tt.args.stanza); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestConcatExecArgs(t *testing.T) {
	type args struct {
		slices [][]string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"concatExecArgsEmpty",
			args{[][]string{{}, {}}},
			[]string{},
		},
		{"concatExecArgsNotEmptyAndEmpty",
			args{[][]string{{"test", "data"}, {}}},
			[]string{"test", "data"},
		},
		{"concatExecArgsEmptyAndNotEmpty",
			args{[][]string{{}, {"test", "data"}}},
			[]string{"test", "data"},
		},
		{"concatExecArgsNotEmpty",
			args{[][]string{{"the", "best"}, {"test", "data"}}},
			[]string{"the", "best", "test", "data"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := concatExecArgs(tt.args.slices); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func fakeSetUpMetricValue(metric *prometheus.GaugeVec, value float64, labels ...string) error {
	return errors.New("сustorm error for test")
}

//nolint:unparam
func templateStanza(walMax, walMin string) stanza {
	return stanza{
		[]archive{
			{databaseID{1, 1}, "13-1", walMax, walMin},
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

//nolint:unparam
func templateStanzaRepoAbsent(walMax, walMin string) stanza {
	return stanza{
		[]archive{
			{databaseID{1, 0}, "13-1", walMax, walMin},
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
				databaseID{1, 0},
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
			{1, 0, 6970977677138971135, "13"},
		},
		"demo",
		[]repo{},
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
