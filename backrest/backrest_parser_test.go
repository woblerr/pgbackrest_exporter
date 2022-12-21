package backrest

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

type mockBackupLastStruct struct {
	mockFull mockStruct
	mockDiff mockStruct
	mockIncr mockStruct
}

var (
	currentUnixTimeForTests = parseDate("2021-07-22 21:00:00").UnixNano()
	mockDataBackupLast      = mockBackupLastStruct{}
)

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
// With '--backrest.database-count' flag.
//
//nolint:dupl
func TestGetBackupMetrics(t *testing.T) {
	type args struct {
		config              string
		configIncludePath   string
		stanzaName          string
		backupData          []backup
		dbData              []db
		backupDBCount       bool
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
		testLastBackups     lastBackupsStruct
	}
	templateMetrics := `# HELP pgbackrest_backup_databases Number of databases in backup.
# TYPE pgbackrest_backup_databases gauge
pgbackrest_backup_databases{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 1
# HELP pgbackrest_backup_delta_bytes Amount of data in the database to actually backup.
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
		name         string
		args         args
		mockTestData mockStruct
	}{
		{
			"getBackupMetrics",
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
					true).Backup,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).DB,
				true,
				setUpMetricValue,
				templateMetrics,
				templateLastBackup(),
			},
			mockStruct{
				`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
					`"max":"000000010000000000000002","min":"000000010000000000000001"}],` +
					`"backup":[{"archive":{"start":"000000010000000000000002","stop":"000000010000000000000002"},` +
					`"backrest":{"format":5,"version":"2.41"},"database":{"id":1,"repo-key":1},` +
					`"database-ref":[{"name":"postgres","oid":13412}],"error":true,"error-list":["base/1/3351"],` +
					`"info":{"delta":24316343,"repository":{"delta":2969512,"size":2969512},"size":24316343},` +
					`"label":"20210614-213200F","lsn":{"start":"0/2000028","stop":"0/2000100"},"prior":null,"reference":null,"timestamp":{"start":1623706320,` +
					`"stop":1623706322},"type":"full"}],"cipher":"none","db":[{"id":1,"repo-key":1,` +
					`"system-id":6970977677138971135,"version":"13"}],"name":"demo","repo":[{"cipher":"none",` +
					`"key":1,"status":{"code":0,"message":"ok"}}],"status":{"code":0,"lock":{"backup":` +
					`{"held":false}},"message":"ok"}}]`,
				"",
				0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			mockData = tt.mockTestData
			execCommand = fakeExecCommand
			defer func() { execCommand = exec.Command }()
			testLastBackups := getBackupMetrics(tt.args.config, tt.args.configIncludePath, tt.args.stanzaName, tt.args.backupData, tt.args.dbData, tt.args.backupDBCount, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaBackupInfoMetric,
				pgbrStanzaBackupDurationMetric,
				pgbrStanzaBackupDatabaseSizeMetric,
				pgbrStanzaBackupDatabaseBackupSizeMetric,
				pgbrStanzaBackupRepoBackupSetSizeMetric,
				pgbrStanzaBackupRepoBackupSizeMetric,
				pgbrStanzaBackupErrorMetric,
				pgbrStanzaBackupDatabasesMetric,
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
//   - pgbackrest_backup_databases
//
// pgBackrest version < 2.41.
//
//nolint:dupl
func TestGetBackupMetricsDBsAbsent(t *testing.T) {
	type args struct {
		config              string
		configIncludePath   string
		stanzaName          string
		backupData          []backup
		dbData              []db
		backupDBCount       bool
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
		name         string
		args         args
		mockTestData mockStruct
	}{
		{
			"getBackupMetrics",
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
					true).Backup,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).DB,
				true,
				setUpMetricValue,
				templateMetrics,
				templateLastBackup(),
			},
			mockStruct{
				"",
				"ERROR: [027]: option 'set' is currently only valid for text output",
				27,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			mockData = tt.mockTestData
			execCommand = fakeExecCommand
			defer func() { execCommand = exec.Command }()
			testLastBackups := getBackupMetrics(tt.args.config, tt.args.configIncludePath, tt.args.stanzaName, tt.args.backupData, tt.args.dbData, tt.args.backupDBCount, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaBackupInfoMetric,
				pgbrStanzaBackupDurationMetric,
				pgbrStanzaBackupDatabaseSizeMetric,
				pgbrStanzaBackupDatabaseBackupSizeMetric,
				pgbrStanzaBackupRepoBackupSetSizeMetric,
				pgbrStanzaBackupRepoBackupSizeMetric,
				pgbrStanzaBackupErrorMetric,
				pgbrStanzaBackupDatabasesMetric,
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
//   - pgbackrest_backup_error_status
//   - pgbackrest_backup_databases
//
// Labels:
//   - lsn_start=""
//   - lsn_stop=""
//
// pgBackrest version < 2.36.
//
//nolint:dupl
func TestGetBackupMetricsErrorAbsent(t *testing.T) {
	type args struct {
		config              string
		configIncludePath   string
		stanzaName          string
		backupData          []backup
		dbData              []db
		backupDBCount       bool
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
		name         string
		args         args
		mockTestData mockStruct
	}{
		{
			"getBackupMetricsErrorAbsent",
			args{
				"",
				"",
				templateStanzaErrorAbsent(
					"000000010000000000000004",
					"000000010000000000000001").Name,
				templateStanzaErrorAbsent(
					"000000010000000000000004",
					"000000010000000000000001").Backup,
				templateStanzaErrorAbsent(
					"000000010000000000000004",
					"000000010000000000000001").DB,
				true,
				setUpMetricValue,
				templateMetrics,
				templateLastBackup(),
			},
			mockStruct{
				"",
				"ERROR: [027]: option 'set' is currently only valid for text output",
				27,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			mockData = tt.mockTestData
			execCommand = fakeExecCommand
			defer func() { execCommand = exec.Command }()
			testLastBackups := getBackupMetrics(tt.args.config, tt.args.configIncludePath, tt.args.stanzaName, tt.args.backupData, tt.args.dbData, tt.args.backupDBCount, tt.args.setUpMetricValueFun, logger)
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
//   - pgbackrest_backup_error_status
//   - pgbackrest_backup_databases
//
// Labels:
//   - repo_key="0"
//   - lsn_start=""
//   - lsn_stop=""
//
// pgBackrest version < v2.32
//
//nolint:dupl
func TestGetBackupMetricsRepoAbsent(t *testing.T) {
	type args struct {
		config              string
		configIncludePath   string
		stanzaName          string
		backupData          []backup
		dbData              []db
		backupDBCount       bool
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
		name         string
		args         args
		mockTestData mockStruct
	}{
		{
			"getBackupMetricsRepoAbsent",
			args{
				"",
				"",
				templateStanzaRepoAbsent(
					"000000010000000000000004",
					"000000010000000000000001").Name,
				templateStanzaRepoAbsent(
					"000000010000000000000004",
					"000000010000000000000001").Backup,
				templateStanzaRepoAbsent(
					"000000010000000000000004",
					"000000010000000000000001").DB,
				false,
				setUpMetricValue,
				templateMetrics,
				templateLastBackup(),
			},
			mockStruct{
				"",
				"ERROR: [027]: option 'set' is currently only valid for text output",
				27,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			mockData = tt.mockTestData
			execCommand = fakeExecCommand
			defer func() { execCommand = exec.Command }()
			testLastBackups := getBackupMetrics(tt.args.config, tt.args.configIncludePath, tt.args.stanzaName, tt.args.backupData, tt.args.dbData, tt.args.backupDBCount, tt.args.setUpMetricValueFun, logger)
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
		config              string
		configIncludePath   string
		stanzaName          string
		backupData          []backup
		dbData              []db
		backupDBCount       bool
		setUpMetricValueFun setUpMetricValueFunType
		errorsCount         int
		debugsCount         int
	}
	tests := []struct {
		name         string
		args         args
		mockTestData mockStruct
	}{
		{
			"getBackupMetricsLogError",
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
					true).Backup,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true).DB,
				true,
				fakeSetUpMetricValue,
				8,
				8,
			},
			mockStruct{
				`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
					`"max":"000000010000000000000002","min":"000000010000000000000001"}],` +
					`"backup":[{"archive":{"start":"000000010000000000000002","stop":"000000010000000000000002"},` +
					`"backrest":{"format":5,"version":"2.41"},"database":{"id":1,"repo-key":1},` +
					`"database-ref":[{"name":"postgres","oid":13412}],"error":true,"error-list":["base/1/3351"],` +
					`"info":{"delta":24316343,"repository":{"delta":2969512,"size":2969512},"size":24316343},` +
					`"label":"20210614-213200F","lsn":{"start":"0/2000028","stop":"0/2000100"},"prior":null,"reference":null,"timestamp":{"start":1623706320,` +
					`"stop":1623706322},"type":"full"}],"cipher":"none","db":[{"id":1,"repo-key":1,` +
					`"system-id":6970977677138971135,"version":"13"}],"name":"demo","repo":[{"cipher":"none",` +
					`"key":1,"status":{"code":0,"message":"ok"}}],"status":{"code":0,"lock":{"backup":` +
					`{"held":false}},"message":"ok"}}]`,
				"",
				0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			mockData = tt.mockTestData
			execCommand = fakeExecCommand
			defer func() { execCommand = exec.Command }()
			out := &bytes.Buffer{}
			lc := log.NewLogfmtLogger(out)
			getBackupMetrics(tt.args.config, tt.args.configIncludePath, tt.args.stanzaName, tt.args.backupData, tt.args.dbData, tt.args.backupDBCount, tt.args.setUpMetricValueFun, lc)
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
// With '--backrest.database-count-latest' flag.
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
	templateMetrics := `# HELP pgbackrest_backup_last_databases Number of databases in the last full, differential or incremental backup.
# TYPE pgbackrest_backup_last_databases gauge
pgbackrest_backup_last_databases{backup_type="diff",stanza="demo"} 2
pgbackrest_backup_last_databases{backup_type="full",stanza="demo"} 1
pgbackrest_backup_last_databases{backup_type="incr",stanza="demo"} 2
# HELP pgbackrest_backup_since_last_completion_seconds Seconds since the last completed full, differential or incremental backup.
# TYPE pgbackrest_backup_since_last_completion_seconds gauge
pgbackrest_backup_since_last_completion_seconds{backup_type="diff",stanza="demo"} 9.223372036854776e+09
pgbackrest_backup_since_last_completion_seconds{backup_type="full",stanza="demo"} 9.223372036854776e+09
pgbackrest_backup_since_last_completion_seconds{backup_type="incr",stanza="demo"} 9.223372036854776e+09
`
	tests := []struct {
		name                   string
		args                   args
		mockTestDataBackupLast mockBackupLastStruct
	}{
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
				templateLastBackupDifferent(),
				true,
				currentUnixTimeForTests,
				setUpMetricValue,
				templateMetrics,
			},
			mockBackupLastStruct{
				mockStruct{
					`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
						`"max":"000000010000000000000010","min":"000000010000000000000001"}],` +
						`"backup":[{"archive":{"start":"000000010000000000000003","stop":"000000010000000000000004"},` +
						`"backrest":{"format":5,"version":"2.41"},"database":{"id":1,"repo-key":1},` +
						`"database-ref":[{"name":"postgres","oid":13412}],"error":false,` +
						`"info":{"delta":32230330,"repository":{"delta":3970793,"size":3970793},"size":32230330},` +
						`"label":"20220926-201857F","link":null,"lsn":{"start":"0/3000028","stop":"0/4000050"},"prior":null,` +
						`"reference":null,"tablespace":null,"timestamp":{"start":1664223537,"stop":1664223540},"type":"full"}],` +
						`"cipher":"none","db":[{"id":1,"repo-key":1,"system-id":7147741414128675215,"version":"13"}],` +
						`"name":"demo","repo":[{"cipher":"none","key":1,"status":{"code":0,"message":"ok"}}],` +
						`"status":{"code":0,"lock":{"backup":{"held":false}},"message":"ok"}}]`,
					"",
					0,
				},
				mockStruct{
					`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
						`"max":"000000010000000000000010","min":"000000010000000000000001"}],` +
						`"backup":[{"archive":{"start":"000000010000000000000005","stop":"000000010000000000000006"},` +
						`"backrest":{"format":5,"version":"2.41"},"database":{"id":1,"repo-key":2},` +
						`"database-ref":[{"name":"postgres","oid":13412},{"name":"test_db","oid":16384}],` +
						`"error":false,"info":{"delta":16657,"repository":{"delta":518,"size":3970802},"size":32230338},` +
						`"label":"20220926-201857F_20220926-201901D","link":null,"lsn":{"start":"0/5000028","stop":"0/6000050"},"prior":"20220926-201857F",` +
						`"reference":["20220926-201857F"],"tablespace":null,"timestamp":{"start":1664223541,"stop":1664223543},"type":"diff"}],` +
						`"cipher":"none","db":[{"id":1,"repo-key":1,"system-id":7147741414128675215,"version":"13"}],` +
						`"name":"demo","repo":[{"cipher":"none","key":1,"status":{"code":0,"message":"ok"}}],` +
						`"status":{"code":0,"lock":{"backup":{"held":false}},"message":"ok"}}]`,
					"",
					0,
				},
				mockStruct{
					`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
						`"max":"000000010000000000000010","min":"000000010000000000000001"}],` +
						`"backup":[{"archive":{"start":"000000010000000000000008","stop":"000000010000000000000008"},` +
						`"backrest":{"format":5,"version":"2.41"},"database":{"id":1,"repo-key":1},` +
						`"database-ref":[{"name":"postgres","oid":13412},{"name":"test_db","oid":16384}],` +
						`"error":false,"info":{"delta":16657,"repository":{"delta":519,"size":3970803},"size":32230338},` +
						`"label":"20220926-201854F_20220926-202454I","link":null,"lsn":{"start":"0/8000028","stop":"0/8000138"},` +
						`"prior":"20220926-201854F","reference":["20220926-201854F"],"tablespace":null,` +
						`"timestamp":{"start":1664223894,"stop":1664223896},"type":"incr"}],` +
						`"cipher":"none","db":[{"id":1,"repo-key":1,"system-id":7147741414128675215,"version":"13"}],` +
						`"name":"demo","repo":[{"cipher":"none","key":1,"status":{"code":0,"message":"ok"}}],` +
						`"status":{"code":0,"lock":{"backup":{"held":false}},"message":"ok"}}]`,
					"",
					0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			mockDataBackupLast = tt.mockTestDataBackupLast
			execCommand = fakeExecCommandSpecificDatabase
			defer func() { execCommand = exec.Command }()
			lc := log.NewNopLogger()
			getBackupLastMetrics(tt.args.config, tt.args.configIncludePath, tt.args.stanzaName, tt.args.lastBackups, tt.args.backupDBCountLatest, tt.args.currentUnixTime, tt.args.setUpMetricValueFun, lc)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaBackupSinceLastCompletionSecondsMetric,
				pgbrStanzaBackupLastDatabasesMetric,
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

// Absent metrics:
//   - pgbackrest_backup_last_databases.
//
// pgBackrest version < v2.41.
func TestGetBackupLastMetricsDBsAbsent(t *testing.T) {
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
	templateMetrics := `# HELP pgbackrest_backup_since_last_completion_seconds Seconds since the last completed full, differential or incremental backup.
# TYPE pgbackrest_backup_since_last_completion_seconds gauge
pgbackrest_backup_since_last_completion_seconds{backup_type="diff",stanza="demo"} 9.223372036854776e+09
pgbackrest_backup_since_last_completion_seconds{backup_type="full",stanza="demo"} 9.223372036854776e+09
pgbackrest_backup_since_last_completion_seconds{backup_type="incr",stanza="demo"} 9.223372036854776e+09
`
	tests := []struct {
		name                   string
		args                   args
		mockTestDataBackupLast mockBackupLastStruct
	}{
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
				templateLastBackupDifferent(),
				true,
				currentUnixTimeForTests,
				setUpMetricValue,
				templateMetrics,
			},
			mockBackupLastStruct{
				mockStruct{
					"",
					"ERROR: [027]: option 'set' is currently only valid for text output",
					27,
				},
				mockStruct{
					"",
					"ERROR: [027]: option 'set' is currently only valid for text output",
					27,
				},
				mockStruct{
					"",
					"ERROR: [027]: option 'set' is currently only valid for text output",
					27,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			mockDataBackupLast = tt.mockTestDataBackupLast
			execCommand = fakeExecCommandSpecificDatabase
			defer func() { execCommand = exec.Command }()
			lc := log.NewNopLogger()
			getBackupLastMetrics(tt.args.config, tt.args.configIncludePath, tt.args.stanzaName, tt.args.lastBackups, tt.args.backupDBCountLatest, tt.args.currentUnixTime, tt.args.setUpMetricValueFun, lc)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaBackupSinceLastCompletionSecondsMetric,
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
		name                   string
		args                   args
		mockTestDataBackupLast mockBackupLastStruct
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
				true,
				currentUnixTimeForTests,
				fakeSetUpMetricValue,
				6,
				6,
			},
			mockBackupLastStruct{
				mockStruct{
					`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
						`"max":"000000010000000000000010","min":"000000010000000000000001"}],` +
						`"backup":[{"archive":{"start":"000000010000000000000003","stop":"000000010000000000000004"},` +
						`"backrest":{"format":5,"version":"2.41"},"database":{"id":1,"repo-key":1},` +
						`"database-ref":[{"name":"postgres","oid":13412}],"error":false,` +
						`"info":{"delta":32230330,"repository":{"delta":3970793,"size":3970793},"size":32230330},` +
						`"label":"20220926-201857F","link":null,"lsn":{"start":"0/3000028","stop":"0/4000050"},"prior":null,` +
						`"reference":null,"tablespace":null,"timestamp":{"start":1664223537,"stop":1664223540},"type":"full"}],` +
						`"cipher":"none","db":[{"id":1,"repo-key":1,"system-id":7147741414128675215,"version":"13"}],` +
						`"name":"demo","repo":[{"cipher":"none","key":1,"status":{"code":0,"message":"ok"}}],` +
						`"status":{"code":0,"lock":{"backup":{"held":false}},"message":"ok"}}]`,
					"",
					0,
				},
				mockStruct{
					`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
						`"max":"000000010000000000000010","min":"000000010000000000000001"}],` +
						`"backup":[{"archive":{"start":"000000010000000000000005","stop":"000000010000000000000006"},` +
						`"backrest":{"format":5,"version":"2.41"},"database":{"id":1,"repo-key":2},` +
						`"database-ref":[{"name":"postgres","oid":13412},{"name":"test_db","oid":16384}],` +
						`"error":false,"info":{"delta":16657,"repository":{"delta":518,"size":3970802},"size":32230338},` +
						`"label":"20220926-201857F_20220926-201901D","link":null,"lsn":{"start":"0/5000028","stop":"0/6000050"},"prior":"20220926-201857F",` +
						`"reference":["20220926-201857F"],"tablespace":null,"timestamp":{"start":1664223541,"stop":1664223543},"type":"diff"}],` +
						`"cipher":"none","db":[{"id":1,"repo-key":1,"system-id":7147741414128675215,"version":"13"}],` +
						`"name":"demo","repo":[{"cipher":"none","key":1,"status":{"code":0,"message":"ok"}}],` +
						`"status":{"code":0,"lock":{"backup":{"held":false}},"message":"ok"}}]`,
					"",
					0,
				},
				mockStruct{
					`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
						`"max":"000000010000000000000010","min":"000000010000000000000001"}],` +
						`"backup":[{"archive":{"start":"000000010000000000000008","stop":"000000010000000000000008"},` +
						`"backrest":{"format":5,"version":"2.41"},"database":{"id":1,"repo-key":1},` +
						`"database-ref":[{"name":"postgres","oid":13412},{"name":"test_db","oid":16384}],` +
						`"error":false,"info":{"delta":16657,"repository":{"delta":519,"size":3970803},"size":32230338},` +
						`"label":"20220926-201854F_20220926-202454I","link":null,"lsn":{"start":"0/8000028","stop":"0/8000138"},` +
						`"prior":"20220926-201854F","reference":["20220926-201854F"],"tablespace":null,` +
						`"timestamp":{"start":1664223894,"stop":1664223896},"type":"incr"}],` +
						`"cipher":"none","db":[{"id":1,"repo-key":1,"system-id":7147741414128675215,"version":"13"}],` +
						`"name":"demo","repo":[{"cipher":"none","key":1,"status":{"code":0,"message":"ok"}}],` +
						`"status":{"code":0,"lock":{"backup":{"held":false}},"message":"ok"}}]`,
					"",
					0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDataBackupLast = tt.mockTestDataBackupLast
			execCommand = fakeExecCommandSpecificDatabase
			defer func() { execCommand = exec.Command }()
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

//nolint:unparam
func templateLastBackupDifferent() lastBackupsStruct {
	return lastBackupsStruct{
		backupStruct{"20220926-201857F", "", time.Unix(1623706322, 0)},
		backupStruct{"20220926-201857F_20220926-201901D", "", time.Unix(1623706322, 0)},
		backupStruct{"20220926-201854F_20220926-202454I", "", time.Unix(1623706322, 0)},
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

func fakeExecCommandSpecificDatabase(command string, args ...string) *exec.Cmd {
	var (
		stdOut, stdErr string
		ecode          int
	)
	cs := []string{"-test.run=TestExecCommandHelper", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	switch {
	case checkBackupType(cs, `D$`):
		stdOut = mockDataBackupLast.mockDiff.mockStdout
		stdErr = mockDataBackupLast.mockDiff.mockStderr
		ecode = mockDataBackupLast.mockDiff.mockExit
	case checkBackupType(cs, `I$`):
		stdOut = mockDataBackupLast.mockIncr.mockStdout
		stdErr = mockDataBackupLast.mockIncr.mockStderr
		ecode = mockDataBackupLast.mockIncr.mockExit
	default:
		stdOut = mockDataBackupLast.mockFull.mockStdout
		stdErr = mockDataBackupLast.mockFull.mockStderr
		ecode = mockDataBackupLast.mockFull.mockExit
	}
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1",
		"STDOUT=" + stdOut,
		"STDERR=" + stdErr,
		"EXIT_STATUS=" + strconv.Itoa(ecode)}
	return cmd
}

func checkBackupType(a []string, regex string) bool {
	for _, n := range a {
		found, err := regexp.MatchString(regex, n)
		if err != nil {
			panic(err)
		}
		if found {
			return true
		}
	}
	return false
}
