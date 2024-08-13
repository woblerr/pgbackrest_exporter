package backrest

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

// All metrics exist and all labels are corrected.
// pgBackrest version = latest.
// With '--backrest.database-count' flag.
// With '--backrest.reference-count' flag.
// The case when the backup is performed with block incremental feature flags.
// Metric pgbackrest_backup_repo_size_bytes is set to 0.
//
//nolint:dupl
func TestGetBackupMetrics(t *testing.T) {
	type args struct {
		stanzaName          string
		referenceCountFlag  bool
		backupData          []backup
		dbData              []db
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
		testLastBackups     lastBackupsStruct
	}
	templateMetrics := `# HELP pgbackrest_backup_annotations Number of annotations in backup.
# TYPE pgbackrest_backup_annotations gauge
pgbackrest_backup_annotations{backup_name="20210607-092423F",backup_type="full",block_incr="y",database_id="1",repo_key="1",stanza="demo"} 1
# HELP pgbackrest_backup_delta_bytes Amount of data in the database to actually backup.
# TYPE pgbackrest_backup_delta_bytes gauge
pgbackrest_backup_delta_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="y",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_backup_duration_seconds Backup duration.
# TYPE pgbackrest_backup_duration_seconds gauge
pgbackrest_backup_duration_seconds{backup_name="20210607-092423F",backup_type="full",block_incr="y",database_id="1",repo_key="1",stanza="demo",start_time="2021-06-07 09:24:23",stop_time="2021-06-07 09:24:26"} 3
# HELP pgbackrest_backup_error_status Backup error status.
# TYPE pgbackrest_backup_error_status gauge
pgbackrest_backup_error_status{backup_name="20210607-092423F",backup_type="full",block_incr="y",database_id="1",repo_key="1",stanza="demo"} 1
# HELP pgbackrest_backup_info Backup info.
# TYPE pgbackrest_backup_info gauge
pgbackrest_backup_info{backrest_ver="2.45",backup_name="20210607-092423F",backup_type="full",block_incr="y",database_id="1",lsn_start="0/2000028",lsn_stop="0/2000100",pg_version="13",prior="",repo_key="1",stanza="demo",wal_start="000000010000000000000002",wal_stop="000000010000000000000002"} 1
# HELP pgbackrest_backup_references Number of references to another backup (backup reference list).
# TYPE pgbackrest_backup_references gauge
pgbackrest_backup_references{backup_name="20210607-092423F",backup_type="full",block_incr="y",database_id="1",ref_backup="diff",repo_key="1",stanza="demo"} 0
pgbackrest_backup_references{backup_name="20210607-092423F",backup_type="full",block_incr="y",database_id="1",ref_backup="full",repo_key="1",stanza="demo"} 0
pgbackrest_backup_references{backup_name="20210607-092423F",backup_type="full",block_incr="y",database_id="1",ref_backup="incr",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_backup_repo_delta_bytes Compressed files size in backup.
# TYPE pgbackrest_backup_repo_delta_bytes gauge
pgbackrest_backup_repo_delta_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="y",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_repo_delta_map_bytes Size of block incremental delta map.
# TYPE pgbackrest_backup_repo_delta_map_bytes gauge
pgbackrest_backup_repo_delta_map_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="y",database_id="1",repo_key="1",stanza="demo"} 12
# HELP pgbackrest_backup_repo_size_bytes Full compressed files size to restore the database from backup.
# TYPE pgbackrest_backup_repo_size_bytes gauge
pgbackrest_backup_repo_size_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="y",database_id="1",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_backup_repo_size_map_bytes Size of block incremental map.
# TYPE pgbackrest_backup_repo_size_map_bytes gauge
pgbackrest_backup_repo_size_map_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="y",database_id="1",repo_key="1",stanza="demo"} 100
# HELP pgbackrest_backup_size_bytes Full uncompressed size of the database.
# TYPE pgbackrest_backup_size_bytes gauge
pgbackrest_backup_size_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="y",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
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
					true,
					false,
					12,
					100,
					0,
					0,
					annotation{"testkey": "testvalue"}).Name,
				true,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					false,
					12,
					100,
					0,
					0,
					annotation{"testkey": "testvalue"}).Backup,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					false,
					12,
					100,
					0,
					0,
					annotation{"testkey": "testvalue"}).DB,
				setUpMetricValue,
				templateMetrics,
				templateLastBackup(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetBackupMetrics()
			testLastBackups := getBackupMetrics(tt.args.stanzaName, tt.args.referenceCountFlag, tt.args.backupData, tt.args.dbData, tt.args.setUpMetricValueFun, logger)
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
				pgbrStanzaBackupReferencesMetric,
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
					"\nVariables do not match, metrics:\n%s\nwant:\n%s",
					tt.args.testText, out.String(),
				)
			}
			if !compareBackupStructs(tt.args.testLastBackups.full, testLastBackups.full) ||
				!compareBackupStructs(tt.args.testLastBackups.diff, testLastBackups.diff) ||
				!compareBackupStructs(tt.args.testLastBackups.incr, testLastBackups.incr) {
				t.Errorf(
					"\nVariables do not match, metrics:\nlastBackups:\n%v\nwant:\n%v",
					tt.args.testLastBackups, testLastBackups,
				)
			}
		})
	}
}

// Metrics with zero values:
//   - pgbackrest_backup_repo_size_map_bytes
//   - pgbackrest_backup_repo_delta_map_bytes
//
// pgBackrest version < 2.44.
//
//nolint:dupl
func TestGetRepoMapMetricsAbsent(t *testing.T) {
	type args struct {
		stanzaName          string
		referenceCountFlag  bool
		backupData          []backup
		dbData              []db
		backupDBCount       bool
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
		testLastBackups     lastBackupsStruct
	}
	templateMetrics := `# HELP pgbackrest_backup_annotations Number of annotations in backup.
# TYPE pgbackrest_backup_annotations gauge
pgbackrest_backup_annotations{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 1
# HELP pgbackrest_backup_delta_bytes Amount of data in the database to actually backup.
# TYPE pgbackrest_backup_delta_bytes gauge
pgbackrest_backup_delta_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_backup_duration_seconds Backup duration.
# TYPE pgbackrest_backup_duration_seconds gauge
pgbackrest_backup_duration_seconds{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo",start_time="2021-06-07 09:24:23",stop_time="2021-06-07 09:24:26"} 3
# HELP pgbackrest_backup_error_status Backup error status.
# TYPE pgbackrest_backup_error_status gauge
pgbackrest_backup_error_status{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 1
# HELP pgbackrest_backup_info Backup info.
# TYPE pgbackrest_backup_info gauge
pgbackrest_backup_info{backrest_ver="2.41",backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",lsn_start="0/2000028",lsn_stop="0/2000100",pg_version="13",prior="",repo_key="1",stanza="demo",wal_start="000000010000000000000002",wal_stop="000000010000000000000002"} 1
# HELP pgbackrest_backup_references Number of references to another backup (backup reference list).
# TYPE pgbackrest_backup_references gauge
pgbackrest_backup_references{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",ref_backup="diff",repo_key="1",stanza="demo"} 0
pgbackrest_backup_references{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",ref_backup="full",repo_key="1",stanza="demo"} 0
pgbackrest_backup_references{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",ref_backup="incr",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_backup_repo_delta_bytes Compressed files size in backup.
# TYPE pgbackrest_backup_repo_delta_bytes gauge
pgbackrest_backup_repo_delta_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_repo_delta_map_bytes Size of block incremental delta map.
# TYPE pgbackrest_backup_repo_delta_map_bytes gauge
pgbackrest_backup_repo_delta_map_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_backup_repo_size_bytes Full compressed files size to restore the database from backup.
# TYPE pgbackrest_backup_repo_size_bytes gauge
pgbackrest_backup_repo_size_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_repo_size_map_bytes Size of block incremental map.
# TYPE pgbackrest_backup_repo_size_map_bytes gauge
pgbackrest_backup_repo_size_map_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_backup_size_bytes Full uncompressed size of the database.
# TYPE pgbackrest_backup_size_bytes gauge
pgbackrest_backup_size_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
`
	tests := []struct {
		name string
		args args
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
				true,
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
				templateLastBackupRepoMapSizesAbsent(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetBackupMetrics()
			testLastBackups := getBackupMetrics(tt.args.stanzaName, tt.args.referenceCountFlag, tt.args.backupData, tt.args.dbData, tt.args.setUpMetricValueFun, logger)
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
				pgbrStanzaBackupReferencesMetric,
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
					"\nVariables do not match, metrics:\n%s\nwant:\n%s",
					tt.args.testText, out.String(),
				)
			}
			if !compareBackupStructs(tt.args.testLastBackups.full, testLastBackups.full) ||
				!compareBackupStructs(tt.args.testLastBackups.diff, testLastBackups.diff) ||
				!compareBackupStructs(tt.args.testLastBackups.incr, testLastBackups.incr) {
				t.Errorf(
					"\nVariables do not match, metrics:\nlastBackups:\n%v\nwant:\n%v",
					tt.args.testLastBackups, testLastBackups,
				)
			}
		})
	}
}

// Metrics with zero values:
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
		referenceCountFlag  bool
		backupData          []backup
		dbData              []db
		backupDBCount       bool
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
		testLastBackups     lastBackupsStruct
	}
	templateMetrics := `# HELP pgbackrest_backup_annotations Number of annotations in backup.
# TYPE pgbackrest_backup_annotations gauge
pgbackrest_backup_annotations{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_backup_delta_bytes Amount of data in the database to actually backup.
# TYPE pgbackrest_backup_delta_bytes gauge
pgbackrest_backup_delta_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_backup_duration_seconds Backup duration.
# TYPE pgbackrest_backup_duration_seconds gauge
pgbackrest_backup_duration_seconds{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo",start_time="2021-06-07 09:24:23",stop_time="2021-06-07 09:24:26"} 3
# HELP pgbackrest_backup_error_status Backup error status.
# TYPE pgbackrest_backup_error_status gauge
pgbackrest_backup_error_status{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 1
# HELP pgbackrest_backup_info Backup info.
# TYPE pgbackrest_backup_info gauge
pgbackrest_backup_info{backrest_ver="2.41",backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",lsn_start="0/2000028",lsn_stop="0/2000100",pg_version="13",prior="",repo_key="1",stanza="demo",wal_start="000000010000000000000002",wal_stop="000000010000000000000002"} 1
# HELP pgbackrest_backup_references Number of references to another backup (backup reference list).
# TYPE pgbackrest_backup_references gauge
pgbackrest_backup_references{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",ref_backup="diff",repo_key="1",stanza="demo"} 0
pgbackrest_backup_references{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",ref_backup="full",repo_key="1",stanza="demo"} 0
pgbackrest_backup_references{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",ref_backup="incr",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_backup_repo_delta_bytes Compressed files size in backup.
# TYPE pgbackrest_backup_repo_delta_bytes gauge
pgbackrest_backup_repo_delta_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_repo_delta_map_bytes Size of block incremental delta map.
# TYPE pgbackrest_backup_repo_delta_map_bytes gauge
pgbackrest_backup_repo_delta_map_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_backup_repo_size_bytes Full compressed files size to restore the database from backup.
# TYPE pgbackrest_backup_repo_size_bytes gauge
pgbackrest_backup_repo_size_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_repo_size_map_bytes Size of block incremental map.
# TYPE pgbackrest_backup_repo_size_map_bytes gauge
pgbackrest_backup_repo_size_map_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_backup_size_bytes Full uncompressed size of the database.
# TYPE pgbackrest_backup_size_bytes gauge
pgbackrest_backup_size_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
`
	tests := []struct {
		name string
		args args
	}{
		{
			"getBackupMetrics",
			args{
				templateStanzaDBsAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					2969514).Name,
				true,
				templateStanzaDBsAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					2969514).Backup,
				templateStanzaDBsAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					2969514).DB,
				true,
				setUpMetricValue,
				templateMetrics,
				templateLastBackupDBsAbsent(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetBackupMetrics()
			testLastBackups := getBackupMetrics(tt.args.stanzaName, tt.args.referenceCountFlag, tt.args.backupData, tt.args.dbData, tt.args.setUpMetricValueFun, logger)
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
				pgbrStanzaBackupReferencesMetric,
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
					"\nVariables do not match, metrics:\n%s\nwant:\n%s",
					tt.args.testText, out.String(),
				)
			}
			if !compareBackupStructs(tt.args.testLastBackups.full, testLastBackups.full) ||
				!compareBackupStructs(tt.args.testLastBackups.diff, testLastBackups.diff) ||
				!compareBackupStructs(tt.args.testLastBackups.incr, testLastBackups.incr) {
				t.Errorf(
					"\nVariables do not match, metrics:\nlastBackups:\n%v\nwant:\n%v",
					tt.args.testLastBackups, testLastBackups,
				)
			}
		})
	}
}

// Metrics with zero values:
//   - pgbackrest_backup_error_status
//   - pgbackrest_backup_repo_size_map_bytes
//   - pgbackrest_backup_repo_delta_map_bytes
//   - pgbackrest_backup_annotations
//
// Labels:
//   - lsn_start="-"
//   - lsn_stop="-"
//
// pgBackrest version < 2.36.
//
//nolint:dupl
func TestGetBackupMetricsErrorAbsent(t *testing.T) {
	type args struct {
		stanzaName          string
		referenceCountFlag  bool
		backupData          []backup
		dbData              []db
		backupDBCount       bool
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
		testLastBackups     lastBackupsStruct
	}
	templateMetrics := `# HELP pgbackrest_backup_annotations Number of annotations in backup.
# TYPE pgbackrest_backup_annotations gauge
pgbackrest_backup_annotations{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_backup_delta_bytes Amount of data in the database to actually backup.
# TYPE pgbackrest_backup_delta_bytes gauge
pgbackrest_backup_delta_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_backup_duration_seconds Backup duration.
# TYPE pgbackrest_backup_duration_seconds gauge
pgbackrest_backup_duration_seconds{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo",start_time="2021-06-07 09:24:23",stop_time="2021-06-07 09:24:26"} 3
# HELP pgbackrest_backup_error_status Backup error status.
# TYPE pgbackrest_backup_error_status gauge
pgbackrest_backup_error_status{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_backup_info Backup info.
# TYPE pgbackrest_backup_info gauge
pgbackrest_backup_info{backrest_ver="2.35",backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",lsn_start="-",lsn_stop="-",pg_version="13",prior="",repo_key="1",stanza="demo",wal_start="000000010000000000000002",wal_stop="000000010000000000000002"} 1
# HELP pgbackrest_backup_references Number of references to another backup (backup reference list).
# TYPE pgbackrest_backup_references gauge
pgbackrest_backup_references{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",ref_backup="diff",repo_key="1",stanza="demo"} 0
pgbackrest_backup_references{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",ref_backup="full",repo_key="1",stanza="demo"} 0
pgbackrest_backup_references{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",ref_backup="incr",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_backup_repo_delta_bytes Compressed files size in backup.
# TYPE pgbackrest_backup_repo_delta_bytes gauge
pgbackrest_backup_repo_delta_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_repo_delta_map_bytes Size of block incremental delta map.
# TYPE pgbackrest_backup_repo_delta_map_bytes gauge
pgbackrest_backup_repo_delta_map_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_backup_repo_size_bytes Full compressed files size to restore the database from backup.
# TYPE pgbackrest_backup_repo_size_bytes gauge
pgbackrest_backup_repo_size_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_repo_size_map_bytes Size of block incremental map.
# TYPE pgbackrest_backup_repo_size_map_bytes gauge
pgbackrest_backup_repo_size_map_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 0
# HELP pgbackrest_backup_size_bytes Full uncompressed size of the database.
# TYPE pgbackrest_backup_size_bytes gauge
pgbackrest_backup_size_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="1",stanza="demo"} 2.4316343e+07
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
					"000000010000000000000001",
					2969514).Name,
				true,
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
				templateLastBackupErrorAbsent(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetBackupMetrics()
			testLastBackups := getBackupMetrics(tt.args.stanzaName, tt.args.referenceCountFlag, tt.args.backupData, tt.args.dbData, tt.args.setUpMetricValueFun, logger)
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
				pgbrStanzaBackupReferencesMetric,
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
					"\nVariables do not match, metrics:\n%s\nwant:\n%s",
					tt.args.testText, out.String(),
				)
			}
			if !compareBackupStructs(tt.args.testLastBackups.full, testLastBackups.full) ||
				!compareBackupStructs(tt.args.testLastBackups.diff, testLastBackups.diff) ||
				!compareBackupStructs(tt.args.testLastBackups.incr, testLastBackups.incr) {
				t.Errorf(
					"\nVariables do not match, metrics:\nlastBackups:\n%v\nwant:\n%v",
					tt.args.testLastBackups, testLastBackups,
				)
			}
		})
	}
}

// Metrics with zero values:
//   - pgbackrest_backup_error_status
//   - pgbackrest_backup_repo_size_map_bytes
//   - pgbackrest_backup_repo_delta_map_bytes
//   - pgbackrest_backup_annotations
//
// Labels:
//   - repo_key="0"
//   - lsn_start="-"
//   - lsn_stop="-"
//
// pgBackrest version < v2.32
//
//nolint:dupl
func TestGetBackupMetricsRepoAbsent(t *testing.T) {
	type args struct {
		stanzaName          string
		referenceCountFlag  bool
		backupData          []backup
		dbData              []db
		backupDBCount       bool
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
		testLastBackups     lastBackupsStruct
	}
	templateMetrics := `# HELP pgbackrest_backup_annotations Number of annotations in backup.
# TYPE pgbackrest_backup_annotations gauge
pgbackrest_backup_annotations{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="0",stanza="demo"} 0
# HELP pgbackrest_backup_delta_bytes Amount of data in the database to actually backup.
# TYPE pgbackrest_backup_delta_bytes gauge
pgbackrest_backup_delta_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="0",stanza="demo"} 2.4316343e+07
# HELP pgbackrest_backup_duration_seconds Backup duration.
# TYPE pgbackrest_backup_duration_seconds gauge
pgbackrest_backup_duration_seconds{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="0",stanza="demo",start_time="2021-06-07 09:24:23",stop_time="2021-06-07 09:24:26"} 3
# HELP pgbackrest_backup_error_status Backup error status.
# TYPE pgbackrest_backup_error_status gauge
pgbackrest_backup_error_status{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="0",stanza="demo"} 0
# HELP pgbackrest_backup_info Backup info.
# TYPE pgbackrest_backup_info gauge
pgbackrest_backup_info{backrest_ver="2.31",backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",lsn_start="-",lsn_stop="-",pg_version="13",prior="",repo_key="0",stanza="demo",wal_start="000000010000000000000002",wal_stop="000000010000000000000002"} 1
# HELP pgbackrest_backup_references Number of references to another backup (backup reference list).
# TYPE pgbackrest_backup_references gauge
pgbackrest_backup_references{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",ref_backup="diff",repo_key="0",stanza="demo"} 0
pgbackrest_backup_references{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",ref_backup="full",repo_key="0",stanza="demo"} 0
pgbackrest_backup_references{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",ref_backup="incr",repo_key="0",stanza="demo"} 0
# HELP pgbackrest_backup_repo_delta_bytes Compressed files size in backup.
# TYPE pgbackrest_backup_repo_delta_bytes gauge
pgbackrest_backup_repo_delta_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="0",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_repo_delta_map_bytes Size of block incremental delta map.
# TYPE pgbackrest_backup_repo_delta_map_bytes gauge
pgbackrest_backup_repo_delta_map_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="0",stanza="demo"} 0
# HELP pgbackrest_backup_repo_size_bytes Full compressed files size to restore the database from backup.
# TYPE pgbackrest_backup_repo_size_bytes gauge
pgbackrest_backup_repo_size_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="0",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_repo_size_map_bytes Size of block incremental map.
# TYPE pgbackrest_backup_repo_size_map_bytes gauge
pgbackrest_backup_repo_size_map_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="0",stanza="demo"} 0
# HELP pgbackrest_backup_size_bytes Full uncompressed size of the database.
# TYPE pgbackrest_backup_size_bytes gauge
pgbackrest_backup_size_bytes{backup_name="20210607-092423F",backup_type="full",block_incr="n",database_id="1",repo_key="0",stanza="demo"} 2.4316343e+07
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
					"000000010000000000000001",
					2969514).Name,
				true,
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
				// Re-use this function, because the fields with the same values from *ErrorAbsent case is returned.
				templateLastBackupErrorAbsent(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetBackupMetrics()
			testLastBackups := getBackupMetrics(tt.args.stanzaName, tt.args.referenceCountFlag, tt.args.backupData, tt.args.dbData, tt.args.setUpMetricValueFun, logger)
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
				pgbrStanzaBackupReferencesMetric,
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
					"\nVariables do not match, metrics:\n%s\nwant:\n%s",
					tt.args.testText, out.String(),
				)
			}
			if !compareBackupStructs(tt.args.testLastBackups.full, testLastBackups.full) ||
				!compareBackupStructs(tt.args.testLastBackups.diff, testLastBackups.diff) ||
				!compareBackupStructs(tt.args.testLastBackups.incr, testLastBackups.incr) {
				t.Errorf(
					"\nVariables do not match, metrics:\nlastBackups:\n%v\nwant:\n%v",
					tt.args.testLastBackups, testLastBackups,
				)
			}
		})
	}
}

func TestGetBackupMetricsErrorsAndDebugs(t *testing.T) {
	type args struct {
		stanzaName          string
		referenceCountFlag  bool
		backupData          []backup
		dbData              []db
		backupDBCount       bool
		setUpMetricValueFun setUpMetricValueFunType
		errorsCount         int
		debugsCount         int
	}
	tests := []struct {
		name string
		args args
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
					false,
					12,
					100,
					0,
					0,
					annotation{"testkey": "testvalue"}).Name,
				true,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					false,
					12,
					100,
					0,
					0,
					annotation{"testkey": "testvalue"}).Backup,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					false,
					12,
					100,
					0,
					0,
					annotation{"testkey": "testvalue"}).DB,
				true,
				fakeSetUpMetricValue,
				14,
				13,
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
				true,
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
				14,
				13,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetBackupMetrics()
			out := &bytes.Buffer{}
			lc := log.NewLogfmtLogger(out)
			getBackupMetrics(tt.args.stanzaName, tt.args.referenceCountFlag, tt.args.backupData, tt.args.dbData, tt.args.setUpMetricValueFun, lc)
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
