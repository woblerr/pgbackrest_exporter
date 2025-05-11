package backrest

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

// All metrics exist and all labels are corrected.
// pgBackrest version = latest.
func TestGetStanzaMetrics(t *testing.T) {
	type args struct {
		stanzaName          string
		stanzaStatus        status
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
			"getStanzaMetricsBackupInProgress",
			args{
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					true,
					12,
					100,
					12345,
					1234,
					annotation{"testkey": "testvalue"}).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					true,
					12,
					100,
					12345,
					1234,
					annotation{"testkey": "testvalue"}).Status,
				setUpMetricValue,
				`# HELP pgbackrest_stanza_backup_compete_bytes Completed size for backup in progress.
# TYPE pgbackrest_stanza_backup_compete_bytes gauge
pgbackrest_stanza_backup_compete_bytes{stanza="demo"} 1234
# HELP pgbackrest_stanza_backup_total_bytes Total size for backup in progress.
# TYPE pgbackrest_stanza_backup_total_bytes gauge
pgbackrest_stanza_backup_total_bytes{stanza="demo"} 12345
# HELP pgbackrest_stanza_lock_status Current stanza lock status.
# TYPE pgbackrest_stanza_lock_status gauge
pgbackrest_stanza_lock_status{stanza="demo"} 1
` + templateMetrics,
			},
		},
		{
			"getStanzaMetricsBackupNotInProgress",
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
					annotation{"testkey": "testvalue"}).Status,
				setUpMetricValue,
				`# HELP pgbackrest_stanza_backup_compete_bytes Completed size for backup in progress.
# TYPE pgbackrest_stanza_backup_compete_bytes gauge
pgbackrest_stanza_backup_compete_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_backup_total_bytes Total size for backup in progress.
# TYPE pgbackrest_stanza_backup_total_bytes gauge
pgbackrest_stanza_backup_total_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_lock_status Current stanza lock status.
# TYPE pgbackrest_stanza_lock_status gauge
pgbackrest_stanza_lock_status{stanza="demo"} 0
` + templateMetrics,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetMetrics()
			getStanzaMetrics(tt.args.stanzaName, tt.args.stanzaStatus, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaStatusMetric,
				pgbrStanzaLockStatusMetric,
				pgbrStanzaBackupInProgressTotalMetric,
				pgbrStanzaBackupInProgressCompleteMetric,
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
				t.Errorf("\nVariables do not match, metrics:\n%s\nwant:\n%s", tt.args.testText, out.String())
			}
		})
	}
}

// pgBackrest version < 2.48.
// Metrics always have 0 value:
//   - pgbackrest_stanza_backup_total_bytes
//   - pgbackrest_stanza_backup_complete_bytes
//
//nolint:dupl
func TestGetStanzaMetricsBackupProgressAbsent(t *testing.T) {
	type args struct {
		stanzaName          string
		stanzaStatus        status
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
	}
	templateMetrics := `# HELP pgbackrest_stanza_backup_compete_bytes Completed size for backup in progress.
# TYPE pgbackrest_stanza_backup_compete_bytes gauge
pgbackrest_stanza_backup_compete_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_backup_total_bytes Total size for backup in progress.
# TYPE pgbackrest_stanza_backup_total_bytes gauge
pgbackrest_stanza_backup_total_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_lock_status Current stanza lock status.
# TYPE pgbackrest_stanza_lock_status gauge
pgbackrest_stanza_lock_status{stanza="demo"} 0
# HELP pgbackrest_stanza_status Current stanza status.
# TYPE pgbackrest_stanza_status gauge
pgbackrest_stanza_status{stanza="demo"} 0
`
	tests := []struct {
		name string
		args args
	}{
		{
			"getStanzaMetricsBackupInProgress",
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
					annotation{"testkey": "testvalue"}).Status,
				setUpMetricValue,
				templateMetrics,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetMetrics()
			getStanzaMetrics(tt.args.stanzaName, tt.args.stanzaStatus, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaStatusMetric,
				pgbrStanzaLockStatusMetric,
				pgbrStanzaBackupInProgressTotalMetric,
				pgbrStanzaBackupInProgressCompleteMetric,
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
				t.Errorf("\nVariables do not match, metrics:\n%s\nwant:\n%s", tt.args.testText, out.String())
			}
		})
	}
}

func TestGetStanzaMetricsErrorsAndDebugs(t *testing.T) {
	type args struct {
		stanzaName          string
		stanzaStatus        status
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
					true,
					true,
					12,
					100,
					12345,
					1234,
					annotation{"testkey": "testvalue"}).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					true,
					12,
					100,
					12345,
					1234,
					annotation{"testkey": "testvalue"}).Status,
				fakeSetUpMetricValue,
				4,
				4,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			lc := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug}))
			getStanzaMetrics(tt.args.stanzaName, tt.args.stanzaStatus, tt.args.setUpMetricValueFun, lc)
			errorsOutputCount := strings.Count(out.String(), "level=ERROR")
			debugsOutputCount := strings.Count(out.String(), "level=DEBUG")
			if tt.args.errorsCount != errorsOutputCount || tt.args.debugsCount != debugsOutputCount {
				t.Errorf("\nVariables do not match:\nerrors=%d, debugs=%d\nwant:\nerrors=%d, debugs=%d",
					tt.args.errorsCount, tt.args.debugsCount,
					errorsOutputCount, debugsOutputCount)
			}
		})
	}
}
