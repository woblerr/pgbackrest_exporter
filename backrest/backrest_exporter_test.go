package backrest

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/exporter-toolkit/web"
)

type mockStruct struct {
	mockStdout string
	mockStderr string
	mockExit   int
}

var (
	logger   = getLogger()
	mockData = mockStruct{}
)

func TestSetPromPortAndPath(t *testing.T) {
	var (
		testFlagsConfig = web.FlagConfig{
			WebListenAddresses: &([]string{":9854"}),
			WebSystemdSocket:   func(i bool) *bool { return &i }(false),
			WebConfigFile:      func(i string) *string { return &i }(""),
		}
		testEndpoint = "/metrics"
	)
	SetPromPortAndPath(testFlagsConfig, testEndpoint)
	if testFlagsConfig.WebListenAddresses != webFlagsConfig.WebListenAddresses ||
		testFlagsConfig.WebSystemdSocket != webFlagsConfig.WebSystemdSocket ||
		testFlagsConfig.WebConfigFile != webFlagsConfig.WebConfigFile ||
		testEndpoint != webEndpoint {
		t.Errorf("\nVariables do not match,\nlistenAddresses: %v, want: %v;\n"+
			"systemSocket: %v, want: %v;\nwebConfig: %v, want: %v;\nendpoint: %s, want: %s",
			ptrToVal(testFlagsConfig.WebListenAddresses), ptrToVal(webFlagsConfig.WebListenAddresses),
			ptrToVal(testFlagsConfig.WebSystemdSocket), ptrToVal(webFlagsConfig.WebSystemdSocket),
			ptrToVal(testFlagsConfig.WebConfigFile), ptrToVal(webFlagsConfig.WebConfigFile),
			testEndpoint, webEndpoint,
		)
	}
}

func TestGetPgBackRestInfo(t *testing.T) {
	type args struct {
		config                         string
		configIncludePath              string
		backupType                     string
		stanzas                        []string
		stanzasExclude                 []string
		backupReferenceCount           bool
		backupDBCount                  bool
		backupDBCountLatest            bool
		verboseWAL                     bool
		backupDBCountParallelProcesses int
	}
	tests := []struct {
		name         string
		args         args
		mockTestData mockStruct
		testText     string
	}{
		{
			"GetPgBackRestInfoGoodDataReturn",
			args{"", "", "", []string{""}, []string{""}, true, true, true, false, 1},
			mockStruct{
				`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
					`"max":"000000010000000000000002","min":"000000010000000000000001"}],` +
					`"backup":[{"archive":{"start":"000000010000000000000002","stop":"000000010000000000000002"},` +
					`"backrest":{"format":5,"version":"2.41"},"database":{"id":1,"repo-key":1},` +
					`"error":false,"info":{"delta":24316343,"repository":{"delta":2969514, "delta-map":12,"size":2969514,"size-map":100},"size":24316343},` +
					`"label":"20210614-213200F","lsn":{"start":"0/2000028","stop":"0/2000100"},"prior":null,"reference":null,"timestamp":{"start":1623706320,` +
					`"stop":1623706322},"type":"full"}],"cipher":"none","db":[{"id":1,"repo-key":1,` +
					`"system-id":6970977677138971135,"version":"13"}],"name":"demo","repo":[{"cipher":"none",` +
					`"key":1,"status":{"code":0,"message":"ok"}}],"status":{"code":0,"lock":{"backup":` +
					`{"held":false}},"message":"ok"}}]`,
				``,
				0,
			},
			""},
		{
			"GetPgBackRestInfoGoodDataReturnWithWarn",
			args{"", "", "", []string{""}, []string{""}, true, true, true, false, 1},
			mockStruct{
				`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
					`"max":"000000010000000000000002","min":"000000010000000000000001"}],` +
					`"backup":[{"archive":{"start":"000000010000000000000002","stop":"000000010000000000000002"},` +
					`"backrest":{"format":5,"version":"2.41"},"database":{"id":1,"repo-key":1},` +
					`"error":false,"info":{"delta":24316343,"repository":{"delta":2969514, "delta-map":12,"size":2969514,"size-map":100},"size":24316343},` +
					`"label":"20210614-213200F","lsn":{"start":"0/2000028","stop":"0/2000100"},"prior":null,"reference":null,"timestamp":{"start":1623706320,` +
					`"stop":1623706322},"type":"full"}],"cipher":"none","db":[{"id":1,"repo-key":1,` +
					`"system-id":6970977677138971135,"version":"13"}],"name":"demo","repo":[{"cipher":"none",` +
					`"key":1,"status":{"code":0,"message":"ok"}}],"status":{"code":0,"lock":{"backup":` +
					`{"held":false}},"message":"ok"}}]`,
				`WARN: environment contains invalid option 'test'`,
				0,
			},
			`msg="pgBackRest message" err="WARN: environment contains invalid option 'test'`},
		{
			"GetPgBackRestInfoBadDataReturn",
			args{"", "", "", []string{""}, []string{""}, false, false, false, false, 1},
			mockStruct{
				``,
				`msg="pgBackRest message" err="ERROR: [029]: missing '=' in key/value at line 9: test"`,
				29,
			},
			`msg="Get data from pgBackRest failed" err="exit status 29`},
		{
			"GetPgBackRestInfoZeroDataReturn",
			args{"", "", "", []string{""}, []string{""}, false, false, false, false, 1},
			mockStruct{
				`[]`,
				``,
				0,
			},
			`msg="No backup data returned"`},
		{
			"GetPgBackRestInfoJsonUnmarshalFail",
			args{"", "", "", []string{""}, []string{""}, false, false, false, false, 1},
			mockStruct{
				`[{}`,
				``,
				0,
			},
			`msg="Parse JSON failed" err="unexpected end of JSON input"`},
		{
			"GetPgBackRestInfoEqualIncludeExcludeLists",
			args{"", "", "", []string{"demo"}, []string{"demo"}, false, false, false, false, 1},
			mockStruct{
				``,
				``,
				0,
			},
			`msg="Stanza is specified in include and exclude lists" stanza=demo`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetMetrics()
			mockData = tt.mockTestData
			execCommand = fakeExecCommand
			defer func() { execCommand = exec.Command }()
			out := &bytes.Buffer{}
			lc := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug}))
			GetPgBackRestInfo(
				tt.args.config,
				tt.args.configIncludePath,
				tt.args.backupType,
				tt.args.stanzas,
				tt.args.stanzasExclude,
				tt.args.backupReferenceCount,
				tt.args.backupDBCount,
				tt.args.backupDBCountLatest,
				tt.args.verboseWAL,
				tt.args.backupDBCountParallelProcesses,
				lc,
			)
			if !strings.Contains(out.String(), tt.testText) {
				t.Errorf("\nVariable do not match:\n%s\nwant:\n%s", tt.testText, out.String())
			}
		})
	}
}

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestExecCommandHelper", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	es := strconv.Itoa(mockData.mockExit)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1",
		"STDOUT=" + mockData.mockStdout,
		"STDERR=" + mockData.mockStderr,
		"EXIT_STATUS=" + es}
	return cmd
}

func TestExecCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%s", os.Getenv("STDOUT"))
	fmt.Fprintf(os.Stderr, "%s", os.Getenv("STDERR"))
	i, _ := strconv.Atoi(os.Getenv("EXIT_STATUS"))
	os.Exit(i)
}

// Set logger for tests.
// If it's necessary to capture the logs output in the test,
// a separate logger is used inside the test.
// The info logging level is used.
func getLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// Helper for displaying web.FlagConfig values test messages.
func ptrToVal[T any](v *T) T {
	return *v
}

func valToPtr[T any](v T) *T {
	return &v
}
