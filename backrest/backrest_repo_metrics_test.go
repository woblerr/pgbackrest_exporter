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
func TestRepoMetrics(t *testing.T) {
	type args struct {
		stanzaName          string
		repoData            *[]repo
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
	}
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
					annotation{"testkey": "testvalue"}).Repo,
				setUpMetricValue,
				`# HELP pgbackrest_repo_status Current repository status.
# TYPE pgbackrest_repo_status gauge
pgbackrest_repo_status{cipher="none",repo_key="1",stanza="demo"} 0
`,
			},
		},
		{
			"getRepoMetricsRepoAbsent",
			args{
				templateStanzaRepoAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					2969514).Name,
				templateStanzaRepoAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					2969514).Repo,
				setUpMetricValue,
				`# HELP pgbackrest_repo_status Current repository status.
# TYPE pgbackrest_repo_status gauge
pgbackrest_repo_status{cipher="none",repo_key="0",stanza="demo"} 0
`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetRepoMetrics()
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
		repoData            *[]repo
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
					annotation{"testkey": "testvalue"}).Repo,
				fakeSetUpMetricValue,
				1,
				1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			lc := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug}))
			getRepoMetrics(tt.args.stanzaName, tt.args.repoData, tt.args.setUpMetricValueFun, lc)
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
