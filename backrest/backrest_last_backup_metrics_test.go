package backrest

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
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
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100).Name,
				templateLastBackupDifferent(),
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
			resetMetrics()
			mockDataBackupLast = tt.mockTestDataBackupLast
			execCommand = fakeExecCommandSpecificDatabase
			defer func() { execCommand = exec.Command }()
			lc := log.NewNopLogger()
			getBackupLastMetrics(tt.args.stanzaName, tt.args.lastBackups, tt.args.currentUnixTime, tt.args.setUpMetricValueFun, lc)
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
		stanzaName          string
		lastBackups         lastBackupsStruct
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
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100).Name,
				templateLastBackupDifferent(),
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
			resetMetrics()
			mockDataBackupLast = tt.mockTestDataBackupLast
			execCommand = fakeExecCommandSpecificDatabase
			defer func() { execCommand = exec.Command }()
			lc := log.NewNopLogger()
			getBackupLastMetrics(tt.args.stanzaName, tt.args.lastBackups, tt.args.currentUnixTime, tt.args.setUpMetricValueFun, lc)
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
		stanzaName          string
		lastBackups         lastBackupsStruct
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
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100).Name,
				templateLastBackup(),
				currentUnixTimeForTests,
				fakeSetUpMetricValue,
				3,
				3,
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
			getBackupLastMetrics(tt.args.stanzaName, tt.args.lastBackups, tt.args.currentUnixTime, tt.args.setUpMetricValueFun, lc)
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

//nolint:unparam
func templateLastBackup() lastBackupsStruct {
	return lastBackupsStruct{
		backupStruct{"20210607-092423F", "full", time.Unix(1623706322, 0)},
		backupStruct{"20210607-092423F", "diff", time.Unix(1623706322, 0)},
		backupStruct{"20210607-092423F", "incr", time.Unix(1623706322, 0)},
	}
}

//nolint:unparam
func templateLastBackupDifferent() lastBackupsStruct {
	return lastBackupsStruct{
		backupStruct{"20220926-201857F", "full", time.Unix(1623706322, 0)},
		backupStruct{"20220926-201857F_20220926-201901D", "diff", time.Unix(1623706322, 0)},
		backupStruct{"20220926-201854F_20220926-202454I", "incr", time.Unix(1623706322, 0)},
	}
}
