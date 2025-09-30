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

func TestGetExporterStatusMetrics(t *testing.T) {
	type args struct {
		stanzaName          string
		getDataStatus       bool
		excludeSpecified    bool
		testText            string
		setUpMetricValueFun setUpMetricValueFunType
	}
	tests := []struct {
		name string
		args args
	}{
		{"GetExporterStatusGood",
			args{
				"test",
				true,
				true,
				`# HELP pgbackrest_exporter_status pgBackRest exporter get data status.
# TYPE pgbackrest_exporter_status gauge
pgbackrest_exporter_status{stanza="test"} 1
`,
				setUpMetricValue,
			},
		},
		{"GetExporterStatusBad",
			args{
				"test",
				false,
				false,
				`# HELP pgbackrest_exporter_status pgBackRest exporter get data status.
# TYPE pgbackrest_exporter_status gauge
pgbackrest_exporter_status{stanza="test"} 0
`,
				setUpMetricValue,
			},
		},
		{"GetExporterStatusAllStanzasExceptExcluded",
			args{
				"",
				true,
				true,
				`# HELP pgbackrest_exporter_status pgBackRest exporter get data status.
# TYPE pgbackrest_exporter_status gauge
pgbackrest_exporter_status{stanza="all-stanzas-except-excluded"} 1
`,
				setUpMetricValue,
			},
		},
		{"GetExporterStatusAllStanzas",
			args{
				"",
				true,
				false,
				`# HELP pgbackrest_exporter_status pgBackRest exporter get data status.
# TYPE pgbackrest_exporter_status gauge
pgbackrest_exporter_status{stanza="all-stanzas"} 1
`,
				setUpMetricValue,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetExporterMetrics()
			getExporterStatusMetrics(tt.args.stanzaName, tt.args.getDataStatus, tt.args.excludeSpecified, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(pgbrExporterStatusMetric)
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

func TestGetExporterStatusErrorsAndDebugs(t *testing.T) {
	type args struct {
		stanzaName          string
		getDataStatus       bool
		excludeSpecified    bool
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
				`test`,
				true,
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
			lc := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug}))
			getExporterStatusMetrics(tt.args.stanzaName, tt.args.getDataStatus, tt.args.excludeSpecified, tt.args.setUpMetricValueFun, lc)
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
