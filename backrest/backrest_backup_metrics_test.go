package backrest

import (
	"bytes"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

// All metrics exist and all labels are corrected.
// pgBackrest version = latest.
// With '--backrest.database-count' flag.
// The case when the backup is performed with block incremental feature flags.
// Absent metrics:
//   - pgbackrest_backup_repo_size_bytes
//
//nolint:dupl
func TestGetBackupMetrics(t *testing.T) {
	type args struct {
		stanzaName          string
		backupData          []backup
		dbData              []db
		backupDBCount       bool
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
		testLastBackups     lastBackupsStruct
	}
	templateMetrics := `# HELP pgbackrest_backup_annotations Number of annotations in backup.
# TYPE pgbackrest_backup_annotations gauge
pgbackrest_backup_annotations{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 1
# HELP pgbackrest_backup_delta_bytes Amount of data in the database to actually backup.
# TYPE pgbackrest_backup_delta_bytes gauge
pgbackrest_backup_delta_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_backup_duration_seconds Backup duration.
# TYPE pgbackrest_backup_duration_seconds gauge
pgbackrest_backup_duration_seconds{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo",start_time="2021-06-07 09:24:23",stop_time="2021-06-07 09:24:26"} 3
# HELP pgbackrest_backup_error_status Backup error status.
# TYPE pgbackrest_backup_error_status gauge
pgbackrest_backup_error_status{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 1
# HELP pgbackrest_backup_info Backup info.
# TYPE pgbackrest_backup_info gauge
pgbackrest_backup_info{backrest_ver="2.45",backup_name="20210607-092423F",backup_type="full",block_incr="y",database_id="1",lsn_start="0/2000028",lsn_stop="0/2000100",pg_version="13",prior="",repo_key="1",stanza="demo",wal_start="000000010000000000000002",wal_stop="000000010000000000000002"} 1
# HELP pgbackrest_backup_repo_delta_bytes Compressed files size in backup.
# TYPE pgbackrest_backup_repo_delta_bytes gauge
pgbackrest_backup_repo_delta_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_repo_delta_map_bytes Size of block incremental delta map.
# TYPE pgbackrest_backup_repo_delta_map_bytes gauge
pgbackrest_backup_repo_delta_map_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 12
# HELP pgbackrest_backup_repo_size_map_bytes Size of block incremental map.
# TYPE pgbackrest_backup_repo_size_map_bytes gauge
pgbackrest_backup_repo_size_map_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 100
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
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100,
					annotation{"testkey": "testvalue"}).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100,
					annotation{"testkey": "testvalue"}).Backup,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100,
					annotation{"testkey": "testvalue"}).DB,
				true,
				setUpMetricValue,
				templateMetrics,
				templateLastBackup(),
			},
			mockStruct{
				`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
					`"max":"000000010000000000000002","min":"000000010000000000000001"}],` +
					`"backup":[{"annotation":{"testkey": "testvalue"},"archive":{"start":"000000010000000000000002","stop":"000000010000000000000002"},` +
					`"backrest":{"format":5,"version":"2.45"},"database":{"id":1,"repo-key":1},` +
					`"database-ref":[{"name":"postgres","oid":13412}],"error":true,"error-list":["base/1/3351"],` +
					`"info":{"delta":24316343,"repository":{"delta":2969512, "delta-map":12,"size-map":100},"size":24316343},` +
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
			resetMetrics()
			mockData = tt.mockTestData
			execCommand = fakeExecCommand
			defer func() { execCommand = exec.Command }()
			testLastBackups := getBackupMetrics(tt.args.stanzaName, tt.args.backupData, tt.args.dbData, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaBackupInfoMetric,
				pgbrStanzaBackupDurationMetric,
				pgbrStanzaBackupDatabaseSizeMetric,
				pgbrStanzaBackupDatabaseBackupSizeMetric,
				pgbrStanzaBackupRepoBackupSetSizeMetric,
				pgbrStanzaBackupRepoBackupSetSizeMapMetric,
				pgbrStanzaBackupRepoBackupSizeMetric,
				pgbrStanzaBackupRepoBackupSizeMapMetric,
				pgbrStanzaBackupErrorMetric,
				pgbrStanzaBackupAnnotationsMetric,
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
//   - pgbackrest_backup_repo_size_map_bytes
//   - pgbackrest_backup_repo_delta_map_bytes
//
// pgBackrest version < 2.44.
//
//nolint:dupl
func TestGetRepoMapMetricsAbsent(t *testing.T) {
	type args struct {
		stanzaName          string
		backupData          []backup
		dbData              []db
		backupDBCount       bool
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
		testLastBackups     lastBackupsStruct
	}
	templateMetrics := `# HELP pgbackrest_backup_annotations Number of annotations in backup.
# TYPE pgbackrest_backup_annotations gauge
pgbackrest_backup_annotations{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 1
# HELP pgbackrest_backup_delta_bytes Amount of data in the database to actually backup.
# TYPE pgbackrest_backup_delta_bytes gauge
pgbackrest_backup_delta_bytes{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_backup_duration_seconds Backup duration.
# TYPE pgbackrest_backup_duration_seconds gauge
pgbackrest_backup_duration_seconds{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo",start_time="2021-06-07 09:24:23",stop_time="2021-06-07 09:24:26"} 3
# HELP pgbackrest_backup_error_status Backup error status.
# TYPE pgbackrest_backup_error_status gauge
pgbackrest_backup_error_status{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 1
# HELP pgbackrest_backup_info Backup info.
# TYPE pgbackrest_backup_info gauge
pgbackrest_backup_info{backrest_ver="2.41",backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",lsn_start="0/2000028",lsn_stop="0/2000100",pg_version="13",prior="",repo_key="1",stanza="demo",wal_start="000000010000000000000002",wal_stop="000000010000000000000002"} 1
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
				templateStanzaRepoMapSizesAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true, 2969514,
					annotation{"testkey": "testvalue"}).Name,
				templateStanzaRepoMapSizesAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true, 2969514,
					annotation{"testkey": "testvalue"}).Backup,
				templateStanzaRepoMapSizesAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true, 2969514,
					annotation{"testkey": "testvalue"}).DB,
				true,
				setUpMetricValue,
				templateMetrics,
				templateLastBackup(),
			},
			mockStruct{
				"",
				"ERROR: [031]: invalid option '--repo1-block'",
				31,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetMetrics()
			mockData = tt.mockTestData
			execCommand = fakeExecCommand
			defer func() { execCommand = exec.Command }()
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
				pgbrStanzaBackupAnnotationsMetric,
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
//   - pgbackrest_backup_repo_size_map_bytes
//   - pgbackrest_backup_repo_delta_map_bytes
//   - pgbackrest_backup_annotations
//
// pgBackrest version < 2.41.
//
//nolint:dupl
func TestGetBackupMetricsDBsAbsent(t *testing.T) {
	type args struct {
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
pgbackrest_backup_duration_seconds{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo",start_time="2021-06-07 09:24:23",stop_time="2021-06-07 09:24:26"} 3
# HELP pgbackrest_backup_error_status Backup error status.
# TYPE pgbackrest_backup_error_status gauge
pgbackrest_backup_error_status{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo"} 1
# HELP pgbackrest_backup_info Backup info.
# TYPE pgbackrest_backup_info gauge
pgbackrest_backup_info{backrest_ver="2.41",backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",lsn_start="0/2000028",lsn_stop="0/2000100",pg_version="13",prior="",repo_key="1",stanza="demo",wal_start="000000010000000000000002",wal_stop="000000010000000000000002"} 1
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
				templateStanzaRepoMapSizesAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					2969514,
					nil).Name,
				templateStanzaRepoMapSizesAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					2969514,
					nil).Backup,
				templateStanzaRepoMapSizesAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					2969514,
					nil).DB,
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
			resetMetrics()
			mockData = tt.mockTestData
			execCommand = fakeExecCommand
			defer func() { execCommand = exec.Command }()
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
//   - pgbackrest_backup_error_status
//   - pgbackrest_backup_databases
//   - pgbackrest_backup_repo_size_map_bytes
//   - pgbackrest_backup_repo_delta_map_bytes
//   - pgbackrest_backup_annotations
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
pgbackrest_backup_duration_seconds{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="1",stanza="demo",start_time="2021-06-07 09:24:23",stop_time="2021-06-07 09:24:26"} 3
# HELP pgbackrest_backup_info Backup info.
# TYPE pgbackrest_backup_info gauge
pgbackrest_backup_info{backrest_ver="2.35",backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",lsn_start="",lsn_stop="",pg_version="13",prior="",repo_key="1",stanza="demo",wal_start="000000010000000000000002",wal_stop="000000010000000000000002"} 1
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
				templateStanzaErrorAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					2969514).Name,
				templateStanzaErrorAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					2969514).Backup,
				templateStanzaErrorAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					2969514).DB,
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
			resetMetrics()
			mockData = tt.mockTestData
			execCommand = fakeExecCommand
			defer func() { execCommand = exec.Command }()
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
//   - pgbackrest_backup_error_status
//   - pgbackrest_backup_databases
//   - pgbackrest_backup_repo_size_map_bytes
//   - pgbackrest_backup_repo_delta_map_bytes
//   - pgbackrest_backup_annotations
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
pgbackrest_backup_duration_seconds{backup_name="20210607-092423F",backup_type="full",database_id="1",repo_key="0",stanza="demo",start_time="2021-06-07 09:24:23",stop_time="2021-06-07 09:24:26"} 3
# HELP pgbackrest_backup_info Backup info.
# TYPE pgbackrest_backup_info gauge
pgbackrest_backup_info{backrest_ver="2.31",backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",lsn_start="",lsn_stop="",pg_version="13",prior="",repo_key="0",stanza="demo",wal_start="000000010000000000000002",wal_stop="000000010000000000000002"} 1
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
				templateStanzaRepoAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					2969514).Name,
				templateStanzaRepoAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					2969514).Backup,
				templateStanzaRepoAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					2969514).DB,
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
			resetMetrics()
			mockData = tt.mockTestData
			execCommand = fakeExecCommand
			defer func() { execCommand = exec.Command }()
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
		// pgBackRest >= 2.45
		// Without backup set size.
		{
			"getBackupMetricsLogError",
			args{
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100,
					annotation{"testkey": "testvalue"}).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100,
					annotation{"testkey": "testvalue"}).Backup,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100,
					annotation{"testkey": "testvalue"}).DB,
				true,
				fakeSetUpMetricValue,
				9,
				9,
			},
			mockStruct{
				`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
					`"max":"000000010000000000000002","min":"000000010000000000000001"}],` +
					`"backup":[{"annotation":{"testkey":"testvalue"},"archive":{"start":"000000010000000000000002","stop":"000000010000000000000002"},` +
					`"backrest":{"format":5,"version":"2.41"},"database":{"id":1,"repo-key":1},` +
					`"database-ref":[{"name":"postgres","oid":13412}],"error":true,"error-list":["base/1/3351"],` +
					`"info":{"delta":24316343,"repository":{"delta":2969512, "delta-map":12,"size-map":100},"size":24316343},` +
					`"label":"20210614-213200F","lsn":{"start":"0/2000028","stop":"0/2000100"},"prior":null,"reference":null,"timestamp":{"start":1623706320,` +
					`"stop":1623706322},"type":"full"}],"cipher":"none","db":[{"id":1,"repo-key":1,` +
					`"system-id":6970977677138971135,"version":"13"}],"name":"demo","repo":[{"cipher":"none",` +
					`"key":1,"status":{"code":0,"message":"ok"}}],"status":{"code":0,"lock":{"backup":` +
					`{"held":false}},"message":"ok"}}]`,
				"",
				0,
			},
		},
		// pgBackrest older than v2.45.
		// Here the version is not important.
		// Getting an error is being tested for backup set size.
		{
			"getBackupMetricsLogErrorWithRepo",
			args{
				templateStanzaRepoMapSizesAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					2969514,
					annotation{"testkey": "testvalue"}).Name,
				templateStanzaRepoMapSizesAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					2969514,
					annotation{"testkey": "testvalue"}).Backup,
				templateStanzaRepoMapSizesAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					2969514,
					annotation{"testkey": "testvalue"}).DB,
				true,
				fakeSetUpMetricValue,
				8,
				8,
			},
			mockStruct{
				`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
					`"max":"000000010000000000000002","min":"000000010000000000000001"}],` +
					`"backup":[{"annotation":{"testkey":"testvalue"},"archive":{"start":"000000010000000000000002","stop":"000000010000000000000002"},` +
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
			resetMetrics()
			mockData = tt.mockTestData
			execCommand = fakeExecCommand
			defer func() { execCommand = exec.Command }()
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
