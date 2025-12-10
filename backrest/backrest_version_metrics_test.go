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

func TestGetBackrestVersionMetrics(t *testing.T) {
	type args struct {
		mockStdout          string
		mockStderr          string
		mockExit            int
		testText            string
		setUpMetricValueFun setUpMetricValueFunType
	}
	tests := []struct {
		name string
		args args
	}{
		{"GetBackrestVersionMetrics",
			args{
				"2057000",
				"",
				0,
				`# HELP pgbackrest_version_info Information about pgBackRest version.
# TYPE pgbackrest_version_info gauge
pgbackrest_version_info 2.057e+06
`,
				setUpMetricValue,
			},
		},
		{"GetBackrestVersionMetricsOldVersion",
			args{
				"pgBackRest 2.40",
				"",
				0,
				`# HELP pgbackrest_version_info Information about pgBackRest version.
# TYPE pgbackrest_version_info gauge
pgbackrest_version_info 0
`,
				setUpMetricValue,
			},
		},
		{"GetBackrestVersionMetricsVersionError",
			args{
				"",
				"pgbackrest: command not found",
				127,
				`# HELP pgbackrest_version_info Information about pgBackRest version.
# TYPE pgbackrest_version_info gauge
pgbackrest_version_info 0
`,
				setUpMetricValue,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetVersionMetrics()
			mockData = mockStruct{tt.args.mockStdout, tt.args.mockStderr, tt.args.mockExit}
			execCommand = fakeExecCommand
			defer func() { execCommand = nil }()
			getBackrestVersionMetrics(tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(pgbrVersionInfoMetric)
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

func TestGetBackrestVersionErrorsAndDebugs(t *testing.T) {
	type args struct {
		mockStdout          string
		mockStderr          string
		mockExit            int
		setUpMetricValueFun setUpMetricValueFunType
		errorsCount         int
		debugsCount         int
	}
	tests := []struct {
		name string
		args args
	}{
		{"GetBackrestVersionLogError",
			args{
				"2057000",
				"",
				0,
				fakeSetUpMetricValue,
				1,
				1,
			},
		},
		{"GetBackrestVersionLogErrorGetData",
			args{
				"",
				"pgbackrest: command not found",
				127,
				setUpMetricValue,
				4,
				1,
			},
		},
		{"GetBackrestVersionLogErrorParse",
			args{
				"invalid version",
				"",
				0,
				setUpMetricValue,
				2,
				1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			lc := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug}))
			mockData = mockStruct{tt.args.mockStdout, tt.args.mockStderr, tt.args.mockExit}
			execCommand = fakeExecCommand
			defer func() { execCommand = nil }()
			getBackrestVersionMetrics(tt.args.setUpMetricValueFun, lc)
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
