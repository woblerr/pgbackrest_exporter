package backrest

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
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
		verbose bool
	}
	tests := []struct {
		name       string
		args       args
		mockStdout string
		mockExit   int
		wantErr    bool
	}{
		{"GetPgBackRestInfoGoodDataReturn",
			args{false},
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
			false},
		{"GetPgBackRestInfoBadDataReturn",
			args{false},
			`Forty two`,
			1,
			true},
		{"GetPgBackRestInfoZeroDataReturn",
			args{false},
			`[]`,
			0,
			false},
		{"GetPgBackRestInfoJsonUnmarshalFail",
			args{false},
			`[{}`,
			0,
			true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStdout = tt.mockStdout
			execCommand = fakeExecCommand
			mockExit = tt.mockExit
			defer func() { execCommand = exec.Command }()
			if err := GetPgBackRestInfo(tt.args.verbose); (err != nil) != tt.wantErr {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", err, tt.wantErr)
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
