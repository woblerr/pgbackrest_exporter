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

var currentUnixTimeForTests = parseDate("2021-07-22 21:00:00").UnixNano()

// All metrics exist and all labels are corrected.
// pgBackrest version = latest.
func TestGetStanzaMetrics(t *testing.T) {
	type args struct {
		stanzaName          string
		stanzaStatusCode    int
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
	}
	templateMetrics := `# HELP pgbackrest_stanza_status Current stanza status.
# TYPE pgbackrest_stanza_status gauge
pgbackrest_stanza_status{stanza="demo"} 0
`
	tests := []struct {
		name string
		args args
	}{
		{
			"getStanzaMetrics",
			args{
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Status.Code,
				setUpMetricValue,
				templateMetrics,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			getStanzaMetrics(tt.args.stanzaName, tt.args.stanzaStatusCode, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(pgbrStanzaStatusMetric)
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
				t.Errorf("\nVariables do not match, metrics:\n%s\nwant:\n%s", tt.args.testText, out.String())
			}
		})
	}
}

func TestGetStanzaMetricsErrorsAndDebugs(t *testing.T) {
	type args struct {
		stanzaName          string
		stanzaStatusCode    int
		setUpMetricValueFun setUpMetricValueFunType
		errorsCount         int
		debugsCount         int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"getStanzaMetricsLogError",
			args{
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Status.Code,
				fakeSetUpMetricValue,
				1,
				1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			lc := log.NewLogfmtLogger(out)
			getStanzaMetrics(tt.args.stanzaName, tt.args.stanzaStatusCode, tt.args.setUpMetricValueFun, lc)
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

// All metrics exist and all labels are corrected.
// pgBackrest version = latest.
func TestRepoMetrics(t *testing.T) {
	type args struct {
		stanzaName          string
		repoData            []repo
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
	}
	templateMetrics := `# HELP pgbackrest_repo_status Current repository status.
# TYPE pgbackrest_repo_status gauge
pgbackrest_repo_status{cipher="none",repo_key="1",stanza="demo"} 0
`
	tests := []struct {
		name string
		args args
	}{
		{
			"getRepoMetrics",
			args{
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Repo,
				setUpMetricValue,
				templateMetrics,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			getRepoMetrics(tt.args.stanzaName, tt.args.repoData, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(pgbrRepoStatusMetric)
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
				t.Errorf("\nVariables do not match, metrics:\n%s\nwant:\n%s", tt.args.testText, out.String())
			}
		})
	}
}

func TestGetRepoMetricsErrorsAndDebugs(t *testing.T) {
	type args struct {
		stanzaName          string
		repoData            []repo
		setUpMetricValueFun setUpMetricValueFunType
		errorsCount         int
		debugsCount         int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"getRepoMetricsLogError",
			args{
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Repo,
				fakeSetUpMetricValue,
				1,
				1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			lc := log.NewLogfmtLogger(out)
			getRepoMetrics(tt.args.stanzaName, tt.args.repoData, tt.args.setUpMetricValueFun, lc)
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

// All metrics exist and all labels are corrected.
// pgBackrest version = latest.
func TestGetBackupMetrics(t *testing.T) {
	type args struct {
		stanzaName          string
		backupData          []backup
		dbData              []db
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
		testLastBackups     lastBackupsStruct
	}
	templateMetrics := `# HELP pgbackrest_backup_delta_bytes Amount of data in the database to actually backup.
# TYPE pgbackrest_backup_delta_bytes gauge
pgbackrest_backup_delta_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_backup_duration_seconds Backup duration.
# TYPE pgbackrest_backup_duration_seconds gauge
pgbackrest_backup_duration_seconds{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo",start_time="2021-06-07 12:24:23",stop_time="2021-06-07 12:24:26"} 3
# HELP pgbackrest_backup_error_status Backup error status.
# TYPE pgbackrest_backup_error_status gauge
pgbackrest_backup_error_status{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 1
# HELP pgbackrest_backup_info Backup info.
# TYPE pgbackrest_backup_info gauge
pgbackrest_backup_info{backrest_ver="2.41",backup_name="20210607-092423F",backup_type="full",database_id="1",lsn_start="0/2000028",lsn_stop="0/2000100",pg_version="13",prior="",repo_key="1",stanza="demo",wal_start="000000010000000000000002",wal_stop="000000010000000000000002"} 1
# HELP pgbackrest_backup_repo_delta_bytes Compressed files size in backup.
# TYPE pgbackrest_backup_repo_delta_bytes gauge
pgbackrest_backup_repo_delta_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_repo_size_bytes Full compressed files size to restore the database from backup.
# TYPE pgbackrest_backup_repo_size_bytes gauge
pgbackrest_backup_repo_size_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_size_bytes Full uncompressed size of the database.
# TYPE pgbackrest_backup_size_bytes gauge
pgbackrest_backup_size_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
`
	tests := []struct {
		name string
		args args
	}{
		{
			"getBackupMetrics",
			args{
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Backup,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).DB,
				setUpMetricValue,
				templateMetrics,
				templateLastBackup(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			testLastBackups := getBackupMetrics(tt.args.stanzaName, tt.args.backupData, tt.args.dbData, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaBackupInfoMetric,
				pgbrStanzaBackupDurationMetric,
				pgbrStanzaBackupDatabaseSizeMetric,
				pgbrStanzaBackupDatabaseBackupSizeMetric,
				pgbrStanzaBackupRepoBackupSetSizeMetric,
				pgbrStanzaBackupRepoBackupSizeMetric,
				pgbrStanzaBackupErrorMetric,
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
			if tt.args.testText != out.String() && !reflect.DeepEqual(tt.args.testLastBackups, testLastBackups) {
				t.Errorf(
					"\nVariables do not match, metrics:\n%s\nwant:\n%s\nlastBackups:\n%v\nwant:\n%v",
					tt.args.testText, out.String(),
					tt.args.testLastBackups, testLastBackups,
				)
			}
		})
	}
}

// Absent metrics:
//	* pgbackrest_backup_error_status
// Labels:
//  * lsn_start=""
//	* lsn_stop=""
// pgBackrest version < 2.36.
func TestGetBackupMetricsErrorAbsent(t *testing.T) {
	type args struct {
		stanzaName          string
		backupData          []backup
		dbData              []db
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
		testLastBackups     lastBackupsStruct
	}
	templateMetrics := `# HELP pgbackrest_backup_delta_bytes Amount of data in the database to actually backup.
# TYPE pgbackrest_backup_delta_bytes gauge
pgbackrest_backup_delta_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_backup_duration_seconds Backup duration.
# TYPE pgbackrest_backup_duration_seconds gauge
pgbackrest_backup_duration_seconds{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo",start_time="2021-06-07 12:24:23",stop_time="2021-06-07 12:24:26"} 3
# HELP pgbackrest_backup_info Backup info.
# TYPE pgbackrest_backup_info gauge
pgbackrest_backup_info{backrest_ver="2.35",backup_name="20210607-092423F",backup_type="full",database_id="1",lsn_start="",lsn_stop="",pg_version="13",prior="",repo_key="1",stanza="demo",wal_start="000000010000000000000002",wal_stop="000000010000000000000002"} 1
# HELP pgbackrest_backup_repo_delta_bytes Compressed files size in backup.
# TYPE pgbackrest_backup_repo_delta_bytes gauge
pgbackrest_backup_repo_delta_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_repo_size_bytes Full compressed files size to restore the database from backup.
# TYPE pgbackrest_backup_repo_size_bytes gauge
pgbackrest_backup_repo_size_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_size_bytes Full uncompressed size of the database.
# TYPE pgbackrest_backup_size_bytes gauge
pgbackrest_backup_size_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
`
	tests := []struct {
		name string
		args args
	}{
		{
			"getBackupMetricsErrorAbsent",
			args{
				templateStanzaErrorAbsent(
					"000000010000000000000004",
					"000000010000000000000001").Name,
				templateStanzaErrorAbsent(
					"000000010000000000000004",
					"000000010000000000000001").Backup,
				templateStanzaErrorAbsent(
					"000000010000000000000004",
					"000000010000000000000001").DB,
				setUpMetricValue,
				templateMetrics,
				templateLastBackup(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			testLastBackups := getBackupMetrics(tt.args.stanzaName, tt.args.backupData, tt.args.dbData, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaBackupInfoMetric,
				pgbrStanzaBackupDurationMetric,
				pgbrStanzaBackupDatabaseSizeMetric,
				pgbrStanzaBackupDatabaseBackupSizeMetric,
				pgbrStanzaBackupRepoBackupSetSizeMetric,
				pgbrStanzaBackupRepoBackupSizeMetric,
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
			if tt.args.testText != out.String() && !reflect.DeepEqual(tt.args.testLastBackups, testLastBackups) {
				t.Errorf(
					"\nVariables do not match, metrics:\n%s\nwant:\n%s\nlastBackups:\n%v\nwant:\n%v",
					tt.args.testText, out.String(),
					tt.args.testLastBackups, testLastBackups,
				)
			}
		})
	}
}

// Absent metrics:
//	* pgbackrest_backup_error_status
// Labels:
// 	* repo_key="0"
//  * lsn_start=""
//	* lsn_stop=""
// pgBackrest version < v2.32
func TestGetBackupMetricsRepoAbsent(t *testing.T) {
	type args struct {
		stanzaName          string
		backupData          []backup
		dbData              []db
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
		testLastBackups     lastBackupsStruct
	}
	templateMetrics := `# HELP pgbackrest_backup_delta_bytes Amount of data in the database to actually backup.
# TYPE pgbackrest_backup_delta_bytes gauge
pgbackrest_backup_delta_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_backup_duration_seconds Backup duration.
# TYPE pgbackrest_backup_duration_seconds gauge
pgbackrest_backup_duration_seconds{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo",start_time="2021-06-07 12:24:23",stop_time="2021-06-07 12:24:26"} 3
# HELP pgbackrest_backup_info Backup info.
# TYPE pgbackrest_backup_info gauge
pgbackrest_backup_info{backrest_ver="2.31",backup_name="20210607-092423F",backup_type="full",database_id="1",lsn_start="",lsn_stop="",pg_version="13",prior="",repo_key="0",stanza="demo",wal_start="000000010000000000000002",wal_stop="000000010000000000000002"} 1
# HELP pgbackrest_backup_repo_delta_bytes Compressed files size in backup.
# TYPE pgbackrest_backup_repo_delta_bytes gauge
pgbackrest_backup_repo_delta_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_repo_size_bytes Full compressed files size to restore the database from backup.
# TYPE pgbackrest_backup_repo_size_bytes gauge
pgbackrest_backup_repo_size_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_size_bytes Full uncompressed size of the database.
# TYPE pgbackrest_backup_size_bytes gauge
pgbackrest_backup_size_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo"} 2.4316343e+07
`
	tests := []struct {
		name string
		args args
	}{
		{
			"getBackupMetricsRepoAbsent",
			args{
				templateStanzaRepoAbsent(
					"000000010000000000000004",
					"000000010000000000000001").Name,
				templateStanzaRepoAbsent(
					"000000010000000000000004",
					"000000010000000000000001").Backup,
				templateStanzaRepoAbsent(
					"000000010000000000000004",
					"000000010000000000000001").DB,
				setUpMetricValue,
				templateMetrics,
				templateLastBackup(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			testLastBackups := getBackupMetrics(tt.args.stanzaName, tt.args.backupData, tt.args.dbData, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaBackupInfoMetric,
				pgbrStanzaBackupDurationMetric,
				pgbrStanzaBackupDatabaseSizeMetric,
				pgbrStanzaBackupDatabaseBackupSizeMetric,
				pgbrStanzaBackupRepoBackupSetSizeMetric,
				pgbrStanzaBackupRepoBackupSizeMetric,
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
			if tt.args.testText != out.String() && !reflect.DeepEqual(tt.args.testLastBackups, testLastBackups) {
				t.Errorf(
					"\nVariables do not match, metrics:\n%s\nwant:\n%s\nlastBackups:\n%v\nwant:\n%v",
					tt.args.testText, out.String(),
					tt.args.testLastBackups, testLastBackups,
				)
			}
		})
	}
}

func TestGetBackupMetricsErrorsAndDebugs(t *testing.T) {
	type args struct {
		stanzaName          string
		backupData          []backup
		dbData              []db
		setUpMetricValueFun setUpMetricValueFunType
		errorsCount         int
		debugsCount         int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"getBackupMetricsLogError",
			args{
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Backup,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).DB,
				fakeSetUpMetricValue,
				7,
				7,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			lc := log.NewLogfmtLogger(out)
			getBackupMetrics(tt.args.stanzaName, tt.args.backupData, tt.args.dbData, tt.args.setUpMetricValueFun, lc)
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

// All metrics exist and all labels are corrected.
// pgBackrest version = latest.
func TestGetBackupLastMetrics(t *testing.T) {
	type args struct {
		config              string
		configIncludePath   string
		stanzaName          string
		lastBackups         lastBackupsStruct
		backupDBCountLatest bool
		currentUnixTime     int64
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
	}
	templateMetrics := `# HELP pgbackrest_backup_diff_since_last_completion_seconds Seconds since the last completed full or differential backup.
# TYPE pgbackrest_backup_diff_since_last_completion_seconds gauge
pgbackrest_backup_diff_since_last_completion_seconds{stanza="demo"} 9.223372036854776e+09
# HELP pgbackrest_backup_full_since_last_completion_seconds Seconds since the last completed full backup.
# TYPE pgbackrest_backup_full_since_last_completion_seconds gauge
pgbackrest_backup_full_since_last_completion_seconds{stanza="demo"} 9.223372036854776e+09
# HELP pgbackrest_backup_incr_since_last_completion_seconds Seconds since the last completed full, differential or incremental backup.
# TYPE pgbackrest_backup_incr_since_last_completion_seconds gauge
pgbackrest_backup_incr_since_last_completion_seconds{stanza="demo"} 9.223372036854776e+09
`
	tests := []struct {
		name string
		args args
	}{
		// Without '--backrest.database-count-latest' flag.
		// TODO:
		// 	 add test for backupDBCountLatest = true
		{
			"getBackupLastMetrics",
			args{
				"",
				"",
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Name,
				templateLastBackup(),
				false,
				currentUnixTimeForTests,
				setUpMetricValue,
				templateMetrics,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			getBackupLastMetrics(tt.args.config, tt.args.configIncludePath, tt.args.stanzaName, tt.args.lastBackups, tt.args.backupDBCountLatest, tt.args.currentUnixTime, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaBackupLastFullMetric,
				pgbrStanzaBackupLastDiffMetric,
				pgbrStanzaBackupLastIncrMetric,
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
				t.Errorf(
					"\nVariables do not match, metrics:\n%s\nwant:\n%s", tt.args.testText, out.String())
			}
		})
	}
}

func TestGetBackupLastMetricsErrorsAndDebugs(t *testing.T) {
	type args struct {
		config              string
		configIncludePath   string
		stanzaName          string
		lastBackups         lastBackupsStruct
		backupDBCountLatest bool
		currentUnixTime     int64
		setUpMetricValueFun setUpMetricValueFunType
		errorsCount         int
		debugsCount         int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"getBackupLastMetricsLogError",
			args{
				"",
				"",
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Name,
				templateLastBackup(),
				false,
				currentUnixTimeForTests,
				fakeSetUpMetricValue,
				3,
				3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			lc := log.NewLogfmtLogger(out)
			getBackupLastMetrics(tt.args.config, tt.args.configIncludePath, tt.args.stanzaName, tt.args.lastBackups, tt.args.backupDBCountLatest, tt.args.currentUnixTime, tt.args.setUpMetricValueFun, lc)
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

// All metrics exist and all labels are corrected.
// pgBackrest version = latest.
func TestGetWALMetrics(t *testing.T) {
	type args struct {
		stanzaName          string
		archiveData         []archive
		dbData              []db
		verboseWAL          bool
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
	}
	templateMetrics := `# HELP pgbackrest_wal_archive_status Current WAL archive status.
# TYPE pgbackrest_wal_archive_status gauge
`
	tests := []struct {
		name string
		args args
	}{
		{
			"getWALMetricsVerboseFalse",
			args{
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Archive,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).DB,
				false,
				setUpMetricValue,
				templateMetrics +
					`pgbackrest_wal_archive_status{database_id="1",pg_version="13",repo_key="1",stanza="demo",wal_max="",wal_min=""} 1` +
					"\n",
			},
		},
		{
			"getWALMetricsVerboseTrue",
			args{
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Archive,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).DB,
				true,
				setUpMetricValue,
				templateMetrics +
					`pgbackrest_wal_archive_status{database_id="1",pg_version="13",repo_key="1",stanza="demo",wal_max="000000010000000000000004",wal_min="000000010000000000000001"} 1` +
					"\n",
			},
		},
		{
			"getWALMetricsWithoutWAL",
			args{
				templateStanza(
					"",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Name,
				templateStanza(
					"",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Archive,
				templateStanza(
					"",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).DB,
				false,
				setUpMetricValue,
				templateMetrics +
					`pgbackrest_wal_archive_status{database_id="1",pg_version="13",repo_key="1",stanza="demo",wal_max="",wal_min=""} 0` +
					"\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			getWALMetrics(tt.args.stanzaName, tt.args.archiveData, tt.args.dbData, tt.args.verboseWAL, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(pgbrWALArchivingMetric)
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
				t.Errorf(
					"\nVariables do not match, metrics:\n%s\nwant:\n%s", tt.args.testText, out.String())
			}
		})
	}
}

func TestGetWALMetricsErrorsAndDebugs(t *testing.T) {
	type args struct {
		stanzaName          string
		archiveData         []archive
		dbData              []db
		verboseWAL          bool
		setUpMetricValueFun setUpMetricValueFunType
		errorsCount         int
		debugsCount         int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"getWALMetricsVerboseFalseLogError",
			args{
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Archive,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).DB,
				false,
				fakeSetUpMetricValue,
				1,
				1,
			},
		},
		{
			"getMetricsVerboseTrueLogError",
			args{
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Archive,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).DB,
				true,
				fakeSetUpMetricValue,
				1,
				1,
			},
		},
		{
			"getMetricsWithoutWalLogError",
			args{
				templateStanza(
					"",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Name,
				templateStanza(
					"",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Archive,
				templateStanza(
					"",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).DB,
				false,
				fakeSetUpMetricValue,
				1,
				1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			lc := log.NewLogfmtLogger(out)
			getWALMetrics(tt.args.stanzaName, tt.args.archiveData, tt.args.dbData, tt.args.verboseWAL, tt.args.setUpMetricValueFun, lc)
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
		{"returnConfigExecArgsNotEmptyConfigIncludePath",
			args{"", "/tmp/pgbackrest/conf.d"},
			[]string{"--config-include-path", "/tmp/pgbackrest/conf.d"},
		},
		{"returnConfigExecArgsNotEmptyConfigAndConfigIncludePath",
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
			if got := returnStanzaExecArgs(tt.args.stanza); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestReturnConfigBackupTypeArgs(t *testing.T) {
	type args struct {
		backupType string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"returnBackupTypeExecArgsEmpty",
			args{""},
			[]string{},
		},
		{"returnBackupTypeExecArgsNotEmpty",
			args{"full"},
			[]string{"--type", "full"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := returnBackupTypeExecArgs(tt.args.backupType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestReturnBackupSetExecArgs(t *testing.T) {
	type args struct {
		backupSetLabel string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"returnBackupSetExecArgsEmpty",
			args{""},
			[]string{},
		},
		{"returnBackupSetExecArgsNotEmpty",
			args{"20210607-092423F"},
			[]string{"--set", "20210607-092423F"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := returnBackupSetExecArgs(tt.args.backupSetLabel); !reflect.DeepEqual(got, tt.want) {
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
		backupLabel   string
		backupType    string
	}
	tests := []struct {
		name string
		args args
		want lastBackupsStruct
	}{
		{"compareLastBackupsFull",
			args{&lastBackups, fullDate, "20210721-000101F", "full"},
			lastBackupsStruct{
				backupStruct{"20210721-000101F", "", fullDate},
				backupStruct{"20210721-000101F", "", fullDate},
				backupStruct{"20210721-000101F", "", fullDate},
			},
		},
		{"compareLastBackupsDiff",
			args{&lastBackups, diffDate, "20210721-000101F_20210721-000501D", "diff"},
			lastBackupsStruct{
				backupStruct{"20210721-000101F", "", fullDate},
				backupStruct{"20210721-000101F_20210721-000501D", "", diffDate},
				backupStruct{"20210721-000101F_20210721-000501D", "", diffDate},
			},
		},
		{"compareLastBackupsIncr",
			args{&lastBackups, incrDate, "20210721-000101F_20210721-001001I", "incr"},
			lastBackupsStruct{
				backupStruct{"20210721-000101F", "", fullDate},
				backupStruct{"20210721-000101F_20210721-000501D", "", diffDate},
				backupStruct{"20210721-000101F_20210721-001001I", "", incrDate},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compareLastBackups(tt.args.backups, tt.args.currentBackup, tt.args.backupLabel, tt.args.backupType)
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
func templateStanza(walMax, walMin string, dbRef []databaseRef, errorStatus bool) stanza {
	var (
		link *[]struct {
			Destination string "json:\"destination\""
			Name        string "json:\"name\""
		}
		tablespace *[]struct {
			Destination string `json:"destination"`
			Name        string `json:"name"`
			OID         int    `json:"oid"`
		}
	)
	return stanza{
		[]archive{
			{databaseID{1, 1}, "13-1", walMax, walMin},
		},
		[]backup{
			{
				struct {
					StartWAL string "json:\"start\""
					StopWAL  string "json:\"stop\""
				}{"000000010000000000000002", "000000010000000000000002"},
				struct {
					Format  int    "json:\"format\""
					Version string "json:\"version\""
				}{5, "2.41"},
				databaseID{1, 1},
				&dbRef,
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
				link,
				struct {
					StartLSN string "json:\"start\""
					StopLSN  string "json:\"stop\""
				}{"0/2000028", "0/2000100"},
				"",
				[]string{""},
				tablespace,
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
func templateStanzaErrorAbsent(walMax, walMin string) stanza {
	var (
		errorStatus *bool
		dbRef       *[]databaseRef
		link        *[]struct {
			Destination string "json:\"destination\""
			Name        string "json:\"name\""
		}
		tablespace *[]struct {
			Destination string `json:"destination"`
			Name        string `json:"name"`
			OID         int    `json:"oid"`
		}
	)
	return stanza{
		[]archive{
			{databaseID{1, 1}, "13-1", walMax, walMin},
		},
		[]backup{
			{
				struct {
					StartWAL string "json:\"start\""
					StopWAL  string "json:\"stop\""
				}{"000000010000000000000002", "000000010000000000000002"},
				struct {
					Format  int    "json:\"format\""
					Version string "json:\"version\""
				}{5, "2.35"},
				databaseID{1, 1},
				dbRef,
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
				link,
				struct {
					StartLSN string "json:\"start\""
					StopLSN  string "json:\"stop\""
				}{"", ""},
				"",
				[]string{""},
				tablespace,
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
	var (
		errorStatus *bool
		dbRef       *[]databaseRef
		link        *[]struct {
			Destination string "json:\"destination\""
			Name        string "json:\"name\""
		}
		tablespace *[]struct {
			Destination string `json:"destination"`
			Name        string `json:"name"`
			OID         int    `json:"oid"`
		}
	)
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
				}{5, "2.31"},
				databaseID{1, 0},
				dbRef,
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
				link,
				struct {
					StartLSN string "json:\"start\""
					StopLSN  string "json:\"stop\""
				}{"", ""},
				"",
				[]string{""},
				tablespace,
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

func TestConvertBoolToFloat64(t *testing.T) {
	type args struct {
		value bool
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{"ConvertBoolToFloat64True",
			args{true},
			1,
		},
		{"ConvertBoolToFloat64False",
			args{false},
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertBoolToFloat64(tt.args.value); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

//nolint:unparam
func templateLastBackup() lastBackupsStruct {
	return lastBackupsStruct{
		backupStruct{"20210607-092423F", "", time.Unix(1623706322, 0)},
		backupStruct{"20210607-092423F", "", time.Unix(1623706322, 0)},
		backupStruct{"20210607-092423F", "", time.Unix(1623706322, 0)},
	}
}

func TestGetParsedSpecificBackupInfoDataErrors(t *testing.T) {
	type args struct {
		config            string
		configIncludePath string
		stanzaName        string
		backupLabel       string
		errorsCount       int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"getParsedSpecificBackupInfoDataErrors",
			args{
				"",
				"",
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).Backup[0].Label,
				2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			lc := log.NewLogfmtLogger(out)
			getParsedSpecificBackupInfoData(tt.args.config, tt.args.configIncludePath, tt.args.stanzaName, tt.args.backupLabel, lc)
			errorsOutputCount := strings.Count(out.String(), "level=error")
			if tt.args.errorsCount != errorsOutputCount {
				t.Errorf("\nVariables do not match:\nerrors=%d, want:\nerrors=%d",
					tt.args.errorsCount, errorsOutputCount)
			}
		})
	}
}
