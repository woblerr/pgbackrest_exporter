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
					annotation{"testkey": "testvalue"}).Status.Code,
				setUpMetricValue,
				templateMetrics,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetMetrics()
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
					annotation{"testkey": "testvalue"}).Status.Code,
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
