package backrest

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

var curretnUnixTimeForTests = parseDate("2021-07-22 21:00:00").UnixNano()

func TestGetMetrics(t *testing.T) {
	type args struct {
		data                stanza
		verbose             bool
		testText            string
		setUpMetricValueFun setUpMetricValueFunType
	}
	templateMetrics := `# HELP pgbackrest_backup_delta_bytes Amount of data in the database to actually backup.
# TYPE pgbackrest_backup_delta_bytes gauge
pgbackrest_backup_delta_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_backup_diff_since_last_completion_seconds Seconds since the last completed full or differential backup.
# TYPE pgbackrest_backup_diff_since_last_completion_seconds gauge
pgbackrest_backup_diff_since_last_completion_seconds{stanza="demo"} 9.223372036854776e+09
# HELP pgbackrest_backup_duration_seconds Backup duration.
# TYPE pgbackrest_backup_duration_seconds gauge
pgbackrest_backup_duration_seconds{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo",start_time="2021-06-07 12:24:23",stop_time="2021-06-07 12:24:26"} 3
# HELP pgbackrest_backup_error_status Backup error status.
# TYPE pgbackrest_backup_error_status gauge
pgbackrest_backup_error_status{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 1
# HELP pgbackrest_backup_full_since_last_completion_seconds Seconds since the last completed full backup.
# TYPE pgbackrest_backup_full_since_last_completion_seconds gauge
pgbackrest_backup_full_since_last_completion_seconds{stanza="demo"} 9.223372036854776e+09
# HELP pgbackrest_backup_incr_since_last_completion_seconds Seconds since the last completed full, differential or incremental backup.
# TYPE pgbackrest_backup_incr_since_last_completion_seconds gauge
pgbackrest_backup_incr_since_last_completion_seconds{stanza="demo"} 9.223372036854776e+09
# HELP pgbackrest_backup_info Backup info.
# TYPE pgbackrest_backup_info gauge
pgbackrest_backup_info{backrest_ver="2.36",backup_name="20210607-092423F",backup_type="full",database_id="1",pg_version="13",prior="",repo_key="1",stanza="demo",wal_start="000000010000000000000002",wal_stop="000000010000000000000002"} 1
# HELP pgbackrest_backup_repo_delta_bytes Compressed files size in backup.
# TYPE pgbackrest_backup_repo_delta_bytes gauge
pgbackrest_backup_repo_delta_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_repo_size_bytes Full compressed files size to restore the database from backup.
# TYPE pgbackrest_backup_repo_size_bytes gauge
pgbackrest_backup_repo_size_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_size_bytes Full uncompressed size of the database.
# TYPE pgbackrest_backup_size_bytes gauge
pgbackrest_backup_size_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_repo_status Current repository status.
# TYPE pgbackrest_repo_status gauge
pgbackrest_repo_status{cipher="none",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_stanza_status Current stanza status.
# TYPE pgbackrest_stanza_status gauge
pgbackrest_stanza_status{stanza="demo"} 0
# HELP pgbackrest_wal_archive_status Current WAL archive status.
# TYPE pgbackrest_wal_archive_status gauge
`
	tests := []struct {
		name string
		args args
	}{
		{"getMetricsVerboseFalse",
			args{
				templateStanza("000000010000000000000004", "000000010000000000000001", true),
				false,
				templateMetrics +
					`pgbackrest_wal_archive_status{database_id="1",pg_version="13",repo_key="1",stanza="demo",wal_max="",wal_min=""} 1` +
					"\n",
				setUpMetricValue,
			},
		},
		{"getMetricsVerboseTrue",
			args{
				templateStanza("000000010000000000000004", "000000010000000000000001", true),
				true,
				templateMetrics +
					`pgbackrest_wal_archive_status{database_id="1",pg_version="13",repo_key="1",stanza="demo",wal_max="000000010000000000000004",wal_min="000000010000000000000001"} 1` +
					"\n",
				setUpMetricValue,
			},
		},
		{"getMetricsWithoutWal",
			args{
				templateStanza("", "000000010000000000000001", true),
				false,
				templateMetrics +
					`pgbackrest_wal_archive_status{database_id="1",pg_version="13",repo_key="1",stanza="demo",wal_max="",wal_min=""} 0` +
					"\n",
				setUpMetricValue,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			getMetrics(tt.args.data, tt.args.verbose, curretnUnixTimeForTests, tt.args.setUpMetricValueFun, logger)
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
				pgbrStanzaBackupErrorMetric,
				pgbrStanzaBackupLastFullMetric,
				pgbrStanzaBackupLastDiffMetric,
				pgbrStanzaBackupLastIncrMetric,
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
	templateMetrics := `# HELP pgbackrest_backup_delta_bytes Amount of data in the database to actually backup.
# TYPE pgbackrest_backup_delta_bytes gauge
pgbackrest_backup_delta_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_backup_diff_since_last_completion_seconds Seconds since the last completed full or differential backup.
# TYPE pgbackrest_backup_diff_since_last_completion_seconds gauge
pgbackrest_backup_diff_since_last_completion_seconds{stanza="demo"} 9.223372036854776e+09
# HELP pgbackrest_backup_duration_seconds Backup duration.
# TYPE pgbackrest_backup_duration_seconds gauge
pgbackrest_backup_duration_seconds{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo",start_time="2021-06-07 12:24:23",stop_time="2021-06-07 12:24:26"} 3
# HELP pgbackrest_backup_error_status Backup error status.
# TYPE pgbackrest_backup_error_status gauge
pgbackrest_backup_error_status{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo"} 0
# HELP pgbackrest_backup_full_since_last_completion_seconds Seconds since the last completed full backup.
# TYPE pgbackrest_backup_full_since_last_completion_seconds gauge
pgbackrest_backup_full_since_last_completion_seconds{stanza="demo"} 9.223372036854776e+09
# HELP pgbackrest_backup_incr_since_last_completion_seconds Seconds since the last completed full, differential or incremental backup.
# TYPE pgbackrest_backup_incr_since_last_completion_seconds gauge
pgbackrest_backup_incr_since_last_completion_seconds{stanza="demo"} 9.223372036854776e+09
# HELP pgbackrest_backup_info Backup info.
# TYPE pgbackrest_backup_info gauge
pgbackrest_backup_info{backrest_ver="2.36",backup_name="20210607-092423F",backup_type="full",database_id="1",pg_version="13",prior="",repo_key="0",stanza="demo",wal_start="000000010000000000000002",wal_stop="000000010000000000000002"} 1
# HELP pgbackrest_backup_repo_delta_bytes Compressed files size in backup.
# TYPE pgbackrest_backup_repo_delta_bytes gauge
pgbackrest_backup_repo_delta_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_repo_size_bytes Full compressed files size to restore the database from backup.
# TYPE pgbackrest_backup_repo_size_bytes gauge
pgbackrest_backup_repo_size_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_size_bytes Full uncompressed size of the database.
# TYPE pgbackrest_backup_size_bytes gauge
pgbackrest_backup_size_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_stanza_status Current stanza status.
# TYPE pgbackrest_stanza_status gauge
pgbackrest_stanza_status{stanza="demo"} 0
# HELP pgbackrest_wal_archive_status Current WAL archive status.
# TYPE pgbackrest_wal_archive_status gauge
`
	tests := []struct {
		name string
		args args
	}{
		{"getMetricsVerboseFalse",
			args{
				templateStanzaRepoAbsent("000000010000000000000004", "000000010000000000000001", false),
				false,
				templateMetrics +
					`pgbackrest_wal_archive_status{database_id="1",pg_version="13",repo_key="0",stanza="demo",wal_max="",wal_min=""} 1` +
					"\n",
				setUpMetricValue,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			getMetrics(tt.args.data, tt.args.verbose, curretnUnixTimeForTests, tt.args.setUpMetricValueFun, logger)
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
				pgbrStanzaBackupErrorMetric,
				pgbrStanzaBackupLastFullMetric,
				pgbrStanzaBackupLastDiffMetric,
				pgbrStanzaBackupLastIncrMetric,
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

func TestGetMetricsErrorAbsent(t *testing.T) {
	type args struct {
		data                stanza
		verbose             bool
		testText            string
		setUpMetricValueFun setUpMetricValueFunType
	}
	templateMetrics := `# HELP pgbackrest_backup_delta_bytes Amount of data in the database to actually backup.
# TYPE pgbackrest_backup_delta_bytes gauge
pgbackrest_backup_delta_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_backup_diff_since_last_completion_seconds Seconds since the last completed full or differential backup.
# TYPE pgbackrest_backup_diff_since_last_completion_seconds gauge
pgbackrest_backup_diff_since_last_completion_seconds{stanza="demo"} 9.223372036854776e+09
# HELP pgbackrest_backup_duration_seconds Backup duration.
# TYPE pgbackrest_backup_duration_seconds gauge
pgbackrest_backup_duration_seconds{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo",start_time="2021-06-07 12:24:23",stop_time="2021-06-07 12:24:26"} 3
# HELP pgbackrest_backup_full_since_last_completion_seconds Seconds since the last completed full backup.
# TYPE pgbackrest_backup_full_since_last_completion_seconds gauge
pgbackrest_backup_full_since_last_completion_seconds{stanza="demo"} 9.223372036854776e+09
# HELP pgbackrest_backup_incr_since_last_completion_seconds Seconds since the last completed full, differential or incremental backup.
# TYPE pgbackrest_backup_incr_since_last_completion_seconds gauge
pgbackrest_backup_incr_since_last_completion_seconds{stanza="demo"} 9.223372036854776e+09
# HELP pgbackrest_backup_info Backup info.
# TYPE pgbackrest_backup_info gauge
pgbackrest_backup_info{backrest_ver="2.34",backup_name="20210607-092423F",backup_type="full",database_id="1",pg_version="13",prior="",repo_key="1",stanza="demo",wal_start="000000010000000000000002",wal_stop="000000010000000000000002"} 1
# HELP pgbackrest_backup_repo_delta_bytes Compressed files size in backup.
# TYPE pgbackrest_backup_repo_delta_bytes gauge
pgbackrest_backup_repo_delta_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_repo_size_bytes Full compressed files size to restore the database from backup.
# TYPE pgbackrest_backup_repo_size_bytes gauge
pgbackrest_backup_repo_size_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_size_bytes Full uncompressed size of the database.
# TYPE pgbackrest_backup_size_bytes gauge
pgbackrest_backup_size_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_repo_status Current repository status.
# TYPE pgbackrest_repo_status gauge
pgbackrest_repo_status{cipher="none",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_stanza_status Current stanza status.
# TYPE pgbackrest_stanza_status gauge
pgbackrest_stanza_status{stanza="demo"} 0
# HELP pgbackrest_wal_archive_status Current WAL archive status.
# TYPE pgbackrest_wal_archive_status gauge
`
	tests := []struct {
		name string
		args args
	}{
		{"getMetricsErrorAbsent",
			args{
				templateStanzaErrorAbsent("000000010000000000000004", "000000010000000000000001"),
				false,
				templateMetrics +
					`pgbackrest_wal_archive_status{database_id="1",pg_version="13",repo_key="1",stanza="demo",wal_max="",wal_min=""} 1` +
					"\n",
				setUpMetricValue,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			getMetrics(tt.args.data, tt.args.verbose, curretnUnixTimeForTests, tt.args.setUpMetricValueFun, logger)
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
				pgbrStanzaBackupErrorMetric,
				pgbrStanzaBackupLastFullMetric,
				pgbrStanzaBackupLastDiffMetric,
				pgbrStanzaBackupLastIncrMetric,
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

func TestGetMetricsErrorsAndDebugs(t *testing.T) {
	type args struct {
		data                stanza
		verbose             bool
		setUpMetricValueFun setUpMetricValueFunType
		errorsCount         int
		debugsCount         int
	}
	tests := []struct {
		name string
		args args
	}{
		{"getMetricsVerboseFalseLogError",
			args{
				templateStanza("000000010000000000000004", "000000010000000000000001", true),
				false,
				fakeSetUpMetricValue,
				13,
				13,
			},
		},
		{"getMetricsVerboseTrueLogError",
			args{
				templateStanza("000000010000000000000004", "000000010000000000000001", true),
				true,
				fakeSetUpMetricValue,
				13,
				13,
			},
		},
		{"getMetricsWithoutWalLogError",
			args{
				templateStanza("", "000000010000000000000001", true),
				false,
				fakeSetUpMetricValue,
				13,
				13,
			},
		},
		{"getMetricsVerboseFalseLogErrorRepoAbsent",
			args{
				templateStanzaRepoAbsent("000000010000000000000004", "000000010000000000000001", false),
				false,
				fakeSetUpMetricValue,
				12,
				12,
			},
		},
		{"getMetricsVerboseTrueLogErrorRepoAbsent",
			args{
				templateStanzaRepoAbsent("000000010000000000000004", "000000010000000000000001", false),
				true,
				fakeSetUpMetricValue,
				12,
				12,
			},
		},
		{"getMetricsWithoutWalLogErrorRepoAbsent",
			args{
				templateStanzaRepoAbsent("", "000000010000000000000001", false),
				false,
				fakeSetUpMetricValue,
				12,
				12,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			lc := log.NewLogfmtLogger(out)
			getMetrics(tt.args.data, tt.args.verbose, curretnUnixTimeForTests, tt.args.setUpMetricValueFun, lc)
			errorsOutputCount := strings.Count(out.String(), "level=error")
			debugssOutputCount := strings.Count(out.String(), "level=debug")
			if tt.args.errorsCount != errorsOutputCount || tt.args.debugsCount != debugssOutputCount {
				t.Errorf("\nVariables do not match:\nerrors=%d, debugs=%d\nwant:\nerrors=%d, debugs=%d",
					tt.args.errorsCount, tt.args.debugsCount,
					errorsOutputCount, debugssOutputCount)
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

func TestCompareLastBackups(t *testing.T) {
	fullDate := parseDate("2021-07-21 00:01:01")
	diffDate := parseDate("2021-07-21 00:05:01")
	incrDate := parseDate("2021-07-21 00:10:01")
	lastBackups := lastBackupsStruct{}
	type args struct {
		backups       *lastBackupsStruct
		currentBackup time.Time
		backupType    string
	}
	tests := []struct {
		name string
		args args
		want lastBackupsStruct
	}{
		{"compareLastBackupsFull",
			args{&lastBackups, fullDate, "full"},
			lastBackupsStruct{fullDate, fullDate, fullDate},
		},
		{"compareLastBackupsDiff",
			args{&lastBackups, diffDate, "diff"},
			lastBackupsStruct{fullDate, diffDate, diffDate},
		},
		{"compareLastBackupsIncr",
			args{&lastBackups, incrDate, "incr"},
			lastBackupsStruct{fullDate, diffDate, incrDate},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compareLastBackups(tt.args.backups, tt.args.currentBackup, tt.args.backupType)
			if !reflect.DeepEqual(*tt.args.backups, tt.want) {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", *tt.args.backups, tt.want)
			}
		})
	}
}

func TestStanzaNotInExclude(t *testing.T) {
	type args struct {
		stanza      string
		listExclude []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"stanzaNotInExcludeEmptyListExclude",
			args{"", []string{""}},
			true},
		{"stanzaNotInExcludeEmptyListExcludeNotEmptyStanza",
			args{"demo", []string{""}},
			true},
		{"stanzaNotInExcludeStanzaNotInExcludeList",
			args{"demo", []string{"demo-test", "test"}},
			true},
		{"stanzaNotInExcludeStanzaInExcludeList",
			args{"demo", []string{"demo", "test"}},
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stanzaNotInExclude(tt.args.stanza, tt.args.listExclude); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func fakeSetUpMetricValue(metric *prometheus.GaugeVec, value float64, labels ...string) error {
	return errors.New("сustorm error for test")
}

//nolint:unparam
func templateStanza(walMax, walMin string, errorStatus bool) stanza {
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
				}{5, "2.36"},
				databaseID{1, 1},
				&errorStatus,
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
func templateStanzaRepoAbsent(walMax, walMin string, errorStatus bool) stanza {
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
				}{5, "2.36"},
				databaseID{1, 0},
				&errorStatus,
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

//nolint:unparam
func templateStanzaErrorAbsent(walMax, walMin string) stanza {
	var errorStatus *bool
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
				errorStatus,
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

func parseDate(value string) time.Time {
	valueReturn, err := time.Parse(layout, value)
	if err != nil {
		panic(err)
	}
	return valueReturn
}

func TestGetExporterMetrics(t *testing.T) {
	type args struct {
		exporterVer         string
		testText            string
		setUpMetricValueFun setUpMetricValueFunType
	}
	templateMetrics := `# HELP pgbackrest_exporter_info Information about pgBackRest exporter.
# TYPE pgbackrest_exporter_info gauge
pgbackrest_exporter_info{version="unknown"} 1
`
	tests := []struct {
		name string
		args args
	}{
		{"GetExporterInfoGood",
			args{
				`unknown`,
				templateMetrics,
				setUpMetricValue,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getExporterMetrics(tt.args.exporterVer, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(pgbrExporterInfoMetric)
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

func TestGetExporterInfoErrorsAndDebugs(t *testing.T) {
	type args struct {
		exporterVer         string
		setUpMetricValueFun setUpMetricValueFunType
		errorsCount         int
		debugsCount         int
	}
	tests := []struct {
		name string
		args args
	}{
		{"GetExporterInfoLogError",
			args{
				`unknown`,
				fakeSetUpMetricValue,
				1,
				1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			logger := log.NewLogfmtLogger(out)
			lc := log.With(logger, level.AllowInfo())
			getExporterMetrics(tt.args.exporterVer, tt.args.setUpMetricValueFun, lc)
			errorsOutputCount := strings.Count(out.String(), "level=error")
			debugsOutputCount := strings.Count(out.String(), "level=debug")
			if tt.args.errorsCount != errorsOutputCount || tt.args.debugsCount != debugsOutputCount {
				t.Errorf("\nVariables do not match:\nerrors=%d, debugs=%d\nwant:\nerrors=%d, debugs=%d",
					tt.args.errorsCount, tt.args.debugsCount,
					errorsOutputCount, debugsOutputCount)
			}
		})
	}
}
