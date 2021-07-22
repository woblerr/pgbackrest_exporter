package backrest

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

var (
	mockStdout string
	mockExit   int
)

func TestSetPromPortandPath(t *testing.T) {
	var (
		testPort    = "9854"
		testEndpoit = "/metrics"
	)
	SetPromPortandPath(testPort, testEndpoit)
	if testPort != promPort || testEndpoit != promEndpoint {
		t.Errorf("\nVariables do not match: %s,\nwant: %s;\nendpoint: %s,\nwant: %s", testPort, promPort, testEndpoit, promEndpoint)
	}
}

func TestGetPgBackRestInfo(t *testing.T) {
	type args struct {
		config            string
		configIncludePath string
		stanzas           []string
		verbose           bool
	}
	tests := []struct {
		name       string
		args       args
		mockStdout string
		mockExit   int
		testText   string
	}{
		{"GetPgBackRestInfoGoodDataReturn",
			args{"", "", []string{""}, false},
			`[{"archive":[{"database":{"id":1,"repo-key":1},"id":"13-1",` +
				`"max":"000000010000000000000002","min":"000000010000000000000001"}],` +
				`"backup":[{"archive":{"start":"000000010000000000000002","stop":"000000010000000000000002"},` +
				`"backrest":{"format":5,"version":"2.34"},"database":{"id":1,"repo-key":1},` +
				`"info":{"delta":24316343,"repository":{"delta":2969512,"size":2969512},"size":24316343},` +
				`"label":"20210614-213200F","prior":null,"reference":null,"timestamp":{"start":1623706320,` +
				`"stop":1623706322},"type":"full"}],"cipher":"none","db":[{"id":1,"repo-key":1,` +
				`"system-id":6970977677138971135,"version":"13"}],"name":"demo","repo":[{"cipher":"none",` +
				`"key":1,"status":{"code":0,"message":"ok"}}],"status":{"code":0,"lock":{"backup":` +
				`{"held":false}},"message":"ok"}}]`,
			0,
			""},
		{"GetPgBackRestInfoBadDataReturn",
			args{"", "", []string{""}, false},
			`Forty two`,
			1,
			"[ERROR] Parse JSON failed, invalid character 'F' looking for beginning of value"},
		{"GetPgBackRestInfoZeroDataReturn",
			args{"", "", []string{""}, false},
			`[]`,
			0,
			"[WARN] No backup data returned"},
		{"GetPgBackRestInfoJsonUnmarshalFail",
			args{"", "", []string{""}, false},
			`[{}`,
			0,
			"[ERROR] Parse JSON failed, unexpected end of JSON input"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMetrics()
			mockStdout = tt.mockStdout
			execCommand = fakeExecCommand
			mockExit = tt.mockExit
			defer func() { execCommand = exec.Command }()
			out := &bytes.Buffer{}
			log.SetOutput(out)
			GetPgBackRestInfo(
				tt.args.config,
				tt.args.configIncludePath,
				tt.args.stanzas,
				tt.args.verbose,
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
	es := strconv.Itoa(mockExit)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1",
		"STDOUT=" + mockStdout,
		"EXIT_STATUS=" + es}
	return cmd
}

func TestExecCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "%s", os.Getenv("STDOUT"))
	i, _ := strconv.Atoi(os.Getenv("EXIT_STATUS"))
	os.Exit(i)
}
