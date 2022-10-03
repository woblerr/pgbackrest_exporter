package backrest

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/prometheus/common/promlog"
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
		testPort          = "9854"
		testEndpoint      = "/metrics"
		testTLSConfigPath = ""
	)
	SetPromPortAndPath(testPort, testEndpoint, testTLSConfigPath)
	if testPort != promPort || testEndpoint != promEndpoint || testTLSConfigPath != promTLSConfigPath {
		t.Errorf("\nVariables do not match,\nport: %s, want: %s;\nendpoint: %s, want: %s;\nconfig: %swant: %s",
			testPort, promPort,
			testEndpoint, promEndpoint,
			testTLSConfigPath, promTLSConfigPath,
		)
	}
}

func TestGetPgBackRestInfo(t *testing.T) {
	type args struct {
		config              string
		configIncludePath   string
		backupType          string
		stanzas             []string
		stanzasExclude      []string
		backupDBCount       bool
		backupDBCountLatest bool
		verboseWAL          bool
	}
	tests := []struct {
		name         string
		args         args
		mockTestData mockStruct
		testText     string
	}{
		{
			"GetPgBackRestInfoGoodDataReturn",
			args{"", "", "", []string{""}, []string{""}, true, true, false},
			mockStruct{
				`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
					`"max":"000000010000000000000002","min":"000000010000000000000001"}],` +
					`"backup":[{"archive":{"start":"000000010000000000000002","stop":"000000010000000000000002"},` +
					`"backrest":{"format":5,"version":"2.41"},"database":{"id":1,"repo-key":1},` +
					`"error":false,"info":{"delta":24316343,"repository":{"delta":2969512,"size":2969512},"size":24316343},` +
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
			args{"", "", "", []string{""}, []string{""}, true, true, false},
			mockStruct{
				`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
					`"max":"000000010000000000000002","min":"000000010000000000000001"}],` +
					`"backup":[{"archive":{"start":"000000010000000000000002","stop":"000000010000000000000002"},` +
					`"backrest":{"format":5,"version":"2.41"},"database":{"id":1,"repo-key":1},` +
					`"error":false,"info":{"delta":24316343,"repository":{"delta":2969512,"size":2969512},"size":24316343},` +
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
			args{"", "", "", []string{""}, []string{""}, false, false, false},
			mockStruct{
				``,
				`msg="pgBackRest message" err="ERROR: [029]: missing '=' in key/value at line 9: test"`,
				29,
			},
			`msg="Get data from pgBackRest failed" err="exit status 29`},
		{
			"GetPgBackRestInfoZeroDataReturn",
			args{"", "", "", []string{""}, []string{""}, false, false, false},
			mockStruct{
				`[]`,
				``,
				0,
			},
			`msg="No backup data returned"`},
		{
			"GetPgBackRestInfoJsonUnmarshalFail",
			args{"", "", "", []string{""}, []string{""}, false, false, false},
			mockStruct{
				`[{}`,
				``,
				0,
			},
			`msg="Parse JSON failed" err="unexpected end of JSON input"`},
		{
			"GetPgBackRestInfoEqualIncludeExcludeLists",
			args{"", "", "", []string{"demo"}, []string{"demo"}, false, false, false},
			mockStruct{
				``,
				``,
				0,
			},
			`msg="Stanza is specified in include and exclude lists" stanza=demo`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			mockData = tt.mockTestData
			execCommand = fakeExecCommand
			defer func() { execCommand = exec.Command }()
			out := &bytes.Buffer{}
			lc := log.NewLogfmtLogger(out)
			GetPgBackRestInfo(
				tt.args.config,
				tt.args.configIncludePath,
				tt.args.backupType,
				tt.args.stanzas,
				tt.args.stanzasExclude,
				tt.args.backupDBCount,
				tt.args.backupDBCountLatest,
				tt.args.verboseWAL,
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
func getLogger() log.Logger {
	var err error
	logLevel := &promlog.AllowedLevel{}
	err = logLevel.Set("info")
	if err != nil {
		panic(err)
	}
	promlogConfig := &promlog.Config{}
	promlogConfig.Level = logLevel
	if err != nil {
		panic(err)
	}
	return promlog.New(promlogConfig)
}
