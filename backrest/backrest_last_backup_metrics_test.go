package backrest

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"testing"

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
func TestGetBackupLastMetrics(t *testing.T) {
	type args struct {
		stanzaName          string
		lastBackups         lastBackupsStruct
		currentUnixTime     int64
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
	}
	templateMetrics := `# HELP pgbackrest_backup_last_annotations Number of annotations in the last full, differential or incremental backup.
# TYPE pgbackrest_backup_last_annotations gauge
pgbackrest_backup_last_annotations{backup_type="diff",block_incr="y",stanza="demo"} 0
pgbackrest_backup_last_annotations{backup_type="full",block_incr="y",stanza="demo"} 1
pgbackrest_backup_last_annotations{backup_type="incr",block_incr="y",stanza="demo"} 0
# HELP pgbackrest_backup_last_delta_bytes Amount of data in the database to actually backup in the last full, differential or incremental backup.
# TYPE pgbackrest_backup_last_delta_bytes gauge
pgbackrest_backup_last_delta_bytes{backup_type="diff",block_incr="y",stanza="demo"} 3.223033e+07
pgbackrest_backup_last_delta_bytes{backup_type="full",block_incr="y",stanza="demo"} 2.4316343e+07
pgbackrest_backup_last_delta_bytes{backup_type="incr",block_incr="y",stanza="demo"} 3.223033e+07
# HELP pgbackrest_backup_last_duration_seconds Backup duration for the last full, differential or incremental backup.
# TYPE pgbackrest_backup_last_duration_seconds gauge
pgbackrest_backup_last_duration_seconds{backup_type="diff",block_incr="y",stanza="demo"} 3
pgbackrest_backup_last_duration_seconds{backup_type="full",block_incr="y",stanza="demo"} 3
pgbackrest_backup_last_duration_seconds{backup_type="incr",block_incr="y",stanza="demo"} 3
# HELP pgbackrest_backup_last_error_status Error status in the last full, differential or incremental backup.
# TYPE pgbackrest_backup_last_error_status gauge
pgbackrest_backup_last_error_status{backup_type="diff",block_incr="y",stanza="demo"} 0
pgbackrest_backup_last_error_status{backup_type="full",block_incr="y",stanza="demo"} 0
pgbackrest_backup_last_error_status{backup_type="incr",block_incr="y",stanza="demo"} 0
# HELP pgbackrest_backup_last_repo_delta_bytes Compressed files size in the last full, differential or incremental backup.
# TYPE pgbackrest_backup_last_repo_delta_bytes gauge
pgbackrest_backup_last_repo_delta_bytes{backup_type="diff",block_incr="y",stanza="demo"} 2.969514e+06
pgbackrest_backup_last_repo_delta_bytes{backup_type="full",block_incr="y",stanza="demo"} 2.969514e+06
pgbackrest_backup_last_repo_delta_bytes{backup_type="incr",block_incr="y",stanza="demo"} 2.969514e+06
# HELP pgbackrest_backup_last_repo_delta_map_bytes Size of block incremental delta map in the last full, differential or incremental backup.
# TYPE pgbackrest_backup_last_repo_delta_map_bytes gauge
pgbackrest_backup_last_repo_delta_map_bytes{backup_type="diff",block_incr="y",stanza="demo"} 12
pgbackrest_backup_last_repo_delta_map_bytes{backup_type="full",block_incr="y",stanza="demo"} 12
pgbackrest_backup_last_repo_delta_map_bytes{backup_type="incr",block_incr="y",stanza="demo"} 12
# HELP pgbackrest_backup_last_repo_size_bytes Full compressed files size to restore the database from the last full, differential or incremental backup.
# TYPE pgbackrest_backup_last_repo_size_bytes gauge
pgbackrest_backup_last_repo_size_bytes{backup_type="diff",block_incr="y",stanza="demo"} 0
pgbackrest_backup_last_repo_size_bytes{backup_type="full",block_incr="y",stanza="demo"} 0
pgbackrest_backup_last_repo_size_bytes{backup_type="incr",block_incr="y",stanza="demo"} 0
# HELP pgbackrest_backup_last_repo_size_map_bytes Size of block incremental map in the last full, differential or incremental backup.
# TYPE pgbackrest_backup_last_repo_size_map_bytes gauge
pgbackrest_backup_last_repo_size_map_bytes{backup_type="diff",block_incr="y",stanza="demo"} 100
pgbackrest_backup_last_repo_size_map_bytes{backup_type="full",block_incr="y",stanza="demo"} 100
pgbackrest_backup_last_repo_size_map_bytes{backup_type="incr",block_incr="y",stanza="demo"} 100
# HELP pgbackrest_backup_last_size_bytes Full uncompressed size of the database in the last full, differential or incremental backup.
# TYPE pgbackrest_backup_last_size_bytes gauge
pgbackrest_backup_last_size_bytes{backup_type="diff",block_incr="y",stanza="demo"} 3.223033e+07
pgbackrest_backup_last_size_bytes{backup_type="full",block_incr="y",stanza="demo"} 2.4316343e+07
pgbackrest_backup_last_size_bytes{backup_type="incr",block_incr="y",stanza="demo"} 3.223033e+07
# HELP pgbackrest_backup_since_last_completion_seconds Seconds since the last completed full, differential or incremental backup.
# TYPE pgbackrest_backup_since_last_completion_seconds gauge
pgbackrest_backup_since_last_completion_seconds{backup_type="diff",block_incr="y",stanza="demo"} 9.223372036854776e+09
pgbackrest_backup_since_last_completion_seconds{backup_type="full",block_incr="y",stanza="demo"} 9.223372036854776e+09
pgbackrest_backup_since_last_completion_seconds{backup_type="incr",block_incr="y",stanza="demo"} 9.223372036854776e+09
`
	tests := []struct {
		name string
		args args
	}{
		{
			"getBackupLastMetrics",
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
				templateLastBackupDifferent(),
				currentUnixTimeForTests,
				setUpMetricValue,
				templateMetrics,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetLastBackupMetrics()
			getBackupLastMetrics(tt.args.stanzaName, tt.args.lastBackups, tt.args.currentUnixTime, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaBackupSinceLastCompletionSecondsMetric,
				pgbrStanzaBackupLastDurationMetric,
				pgbrStanzaBackupLastDatabaseSizeMetric,
				pgbrStanzaBackupLastDatabaseBackupSizeMetric,
				pgbrStanzaBackupLastRepoBackupSetSizeMetric,
				pgbrStanzaBackupLastRepoBackupSetSizeMapMetric,
				pgbrStanzaBackupLastRepoBackupSizeMetric,
				pgbrStanzaBackupLastRepoBackupSizeMapMetric,
				pgbrStanzaBackupLastErrorMetric,
				pgbrStanzaBackupLastAnnotationsMetric,
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

func TestGetBackupLastDBCountMetrics(t *testing.T) {
	type args struct {
		config              string
		configIncludePath   string
		stanzaName          string
		lastBackups         lastBackupsStruct
		currentUnixTime     int64
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
	}
	tests := []struct {
		name                   string
		args                   args
		mockTestDataBackupLast mockBackupLastStruct
	}{
		// All metrics exist and all labels are corrected.
		// pgBackrest version = latest.
		{
			"getBackupLastMetricsDatabases",
			args{
				"",
				"",
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
				templateLastBackup(),
				currentUnixTimeForTests,
				setUpMetricValue,
				`# HELP pgbackrest_backup_last_databases Number of databases in the last full, differential or incremental backup.
# TYPE pgbackrest_backup_last_databases gauge
pgbackrest_backup_last_databases{backup_type="diff",block_incr="y",stanza="demo"} 1
pgbackrest_backup_last_databases{backup_type="full",block_incr="y",stanza="demo"} 1
pgbackrest_backup_last_databases{backup_type="incr",block_incr="y",stanza="demo"} 1
`,
			},
			mockBackupLastStruct{
				mockStruct{
					`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
						`"max":"000000010000000000000010","min":"000000010000000000000001"}],` +
						`"backup":[{"annotation":{"testkey":"testvalue"},"archive":{"start":"000000010000000000000003","stop":"000000010000000000000004"},` +
						`"backrest":{"format":5,"version":"2.48"},"database":{"id":1,"repo-key":1},` +
						`"database-ref":[{"name":"postgres","oid":13412}],"error":true,` +
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
						`"backup":[{"annotation":{"testkey":"testvalue"},"archive":{"start":"000000010000000000000003","stop":"000000010000000000000004"},` +
						`"backrest":{"format":5,"version":"2.48"},"database":{"id":1,"repo-key":1},` +
						`"database-ref":[{"name":"postgres","oid":13412}],"error":true,` +
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
						`"backup":["annotation":{"testkey":"testvalue"},"archive":{"start":"000000010000000000000003","stop":"000000010000000000000004"},` +
						`"backrest":{"format":5,"version":"2.48"},"database":{"id":1,"repo-key":1},` +
						`"database-ref":[{"name":"postgres","oid":13412}],"error":true,` +
						`"info":{"delta":32230330,"repository":{"delta":3970793,"size":3970793},"size":32230330},` +
						`"label":"20220926-201857F","link":null,"lsn":{"start":"0/3000028","stop":"0/4000050"},"prior":null,` +
						`"reference":null,"tablespace":null,"timestamp":{"start":1664223537,"stop":1664223540},"type":"full"}],` +
						`"cipher":"none","db":[{"id":1,"repo-key":1,"system-id":7147741414128675215,"version":"13"}],` +
						`"name":"demo","repo":[{"cipher":"none","key":1,"status":{"code":0,"message":"ok"}}],` +
						`"status":{"code":0,"lock":{"backup":{"held":false}},"message":"ok"}}]`,
					"",
					0,
				},
			},
		},
		// Absent metrics:
		//   - pgbackrest_backup_last_databases.
		//
		// pgBackrest version < v2.41.
		{
			"getBackupLastMetricsDatabasesAbsent",
			args{
				"",
				"",
				templateStanzaDBsAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					2969514).Name,
				templateLastBackupDBsAbsent(),
				currentUnixTimeForTests,
				setUpMetricValue,
				`# HELP pgbackrest_backup_last_databases Number of databases in the last full, differential or incremental backup.
# TYPE pgbackrest_backup_last_databases gauge
pgbackrest_backup_last_databases{backup_type="diff",block_incr="n",stanza="demo"} 0
pgbackrest_backup_last_databases{backup_type="full",block_incr="n",stanza="demo"} 0
pgbackrest_backup_last_databases{backup_type="incr",block_incr="n",stanza="demo"} 0
`,
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
			resetLastBackupMetrics()
			mockDataBackupLast = tt.mockTestDataBackupLast
			execCommand = fakeExecCommandSpecificDatabase
			defer func() { execCommand = exec.Command }()
			lc := slog.New(slog.NewTextHandler(os.Stdout, nil))
			getBackupLastDBCountMetrics(tt.args.config, tt.args.configIncludePath, tt.args.stanzaName, tt.args.lastBackups, tt.args.setUpMetricValueFun, lc)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
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
