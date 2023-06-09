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
					true,
					12,
					100).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100).Archive,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100).DB,
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
					true,
					12,
					100).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100).Archive,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100).DB,
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
					true,
					12,
					100).Name,
				templateStanza(
					"",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100).Archive,
				templateStanza(
					"",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100).DB,
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
			resetMetrics()
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
					true,
					12,
					100).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100).Archive,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100).DB,
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
					true,
					12,
					100).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100).Archive,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100).DB,
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
					true,
					12,
					100).Name,
				templateStanza(
					"",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100).Archive,
				templateStanza(
					"",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100).DB,
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
