package backrest

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

func TestGetPGVersion(t *testing.T) {
	type args struct {
		id      int
		repoKey int
		dbList  []db
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"getPGVersionSame",
			args{1, 1, []db{{1, 1, 6970977677138971135, "13"}}},
			"13",
		},
		{"getPGVersionDiff",
			args{1, 5, []db{{1, 1, 6970977677138971135, "13"}}},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPGVersion(tt.args.id, tt.args.repoKey, tt.args.dbList); got != tt.want {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestSetUpMetricValue(t *testing.T) {
	type args struct {
		metric *prometheus.GaugeVec
		value  float64
		labels []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"setUpMetricValueError",
			args{pgbrStanzaStatusMetric, 0, []string{"demo", "bad"}},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := setUpMetricValue(tt.args.metric, tt.args.value, tt.args.labels...); (err != nil) != tt.wantErr {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestReturnDefaultExecArgs(t *testing.T) {
	testArgs := []string{"info", "--output", "json"}
	defaultArgs := returnDefaultExecArgs()
	if !reflect.DeepEqual(testArgs, defaultArgs) {
		t.Errorf("\nVariables do not match: %s,\nwant: %s", testArgs, defaultArgs)
	}
}

func TestReturnConfigExecArgs(t *testing.T) {
	type args struct {
		config            string
		configIncludePath string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"returnConfigExecArgsEmpty",
			args{"", ""},
			[]string{},
		},
		{"returnConfigExecArgsNotEmptyConfig",
			args{"/tmp/pgbackrest.conf", ""},
			[]string{"--config", "/tmp/pgbackrest.conf"},
		},
		{"returnConfigExecArgsNotEmptyConfigIncludePath",
			args{"", "/tmp/pgbackrest/conf.d"},
			[]string{"--config-include-path", "/tmp/pgbackrest/conf.d"},
		},
		{"returnConfigExecArgsNotEmptyConfigAndConfigIncludePath",
			args{"/tmp/pgbackrest.conf", "/tmp/pgbackrest/conf.d"},
			[]string{"--config", "/tmp/pgbackrest.conf", "--config-include-path", "/tmp/pgbackrest/conf.d"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := returnConfigExecArgs(tt.args.config, tt.args.configIncludePath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestReturnConfigStanzaArgs(t *testing.T) {
	type args struct {
		stanza string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"returnStanzaExecArgsEmpty",
			args{""},
			[]string{},
		},
		{"returnStanzaExecArgsNotEmpty",
			args{"demo"},
			[]string{"--stanza", "demo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := returnStanzaExecArgs(tt.args.stanza); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestReturnConfigBackupTypeArgs(t *testing.T) {
	type args struct {
		backupType string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"returnBackupTypeExecArgsEmpty",
			args{""},
			[]string{},
		},
		{"returnBackupTypeExecArgsNotEmpty",
			args{"full"},
			[]string{"--type", "full"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := returnBackupTypeExecArgs(tt.args.backupType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestReturnBackupSetExecArgs(t *testing.T) {
	type args struct {
		backupSetLabel string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"returnBackupSetExecArgsEmpty",
			args{""},
			[]string{},
		},
		{"returnBackupSetExecArgsNotEmpty",
			args{"20210607-092423F"},
			[]string{"--set", "20210607-092423F"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := returnBackupSetExecArgs(tt.args.backupSetLabel); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestConcatExecArgs(t *testing.T) {
	type args struct {
		slices [][]string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"concatExecArgsEmpty",
			args{[][]string{{}, {}}},
			[]string{},
		},
		{"concatExecArgsNotEmptyAndEmpty",
			args{[][]string{{"test", "data"}, {}}},
			[]string{"test", "data"},
		},
		{"concatExecArgsEmptyAndNotEmpty",
			args{[][]string{{}, {"test", "data"}}},
			[]string{"test", "data"},
		},
		{"concatExecArgsNotEmpty",
			args{[][]string{{"the", "best"}, {"test", "data"}}},
			[]string{"the", "best", "test", "data"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := concatExecArgs(tt.args.slices); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestCompareLastBackups(t *testing.T) {
	var (
		size    *int64
		archive struct {
			StartWAL string "json:\"start\""
			StopWAL  string "json:\"stop\""
		}
		backrestInfo struct {
			Format  int    "json:\"format\""
			Version string "json:\"version\""
		}
		link *[]struct {
			Destination string "json:\"destination\""
			Name        string "json:\"name\""
		}
		tablespace *[]struct {
			Destination string `json:"destination"`
			Name        string `json:"name"`
			OID         int    `json:"oid"`
		}
		lsn struct {
			StartLSN string "json:\"start\""
			StopLSN  string "json:\"stop\""
		}
	)
	fullDate := parseDate("2021-07-21 00:01:04")
	diffDate := parseDate("2021-07-21 00:05:04")
	incrDate := parseDate("2021-07-21 00:10:04")
	backuptTestRepoDeltaMap := valToPtr(int64(12))
	backupTestRepoSizeMap := valToPtr(int64(100))
	backupTestAnnotation := valToPtr(annotation{"testkey": "testvalue"})
	backupTestError := valToPtr(false)
	lastBackups := initLastBackupStruct()
	type args struct {
		backups    *lastBackupsStruct
		backupTest backup
		backupIncr string
	}
	tests := []struct {
		name string
		args args
		want lastBackupsStruct
	}{
		{"compareLastBackupsFull",
			args{
				&lastBackups,
				backup{
					backupTestAnnotation,
					archive,
					backrestInfo,
					databaseID{1, 1},
					nil,
					backupTestError,
					backupInfo{
						24316343,
						struct {
							Delta    int64  "json:\"delta\""
							DeltaMap *int64 "json:\"delta-map\""
							Size     *int64 "json:\"size\""
							SizeMap  *int64 "json:\"size-map\""
						}{2969514, backuptTestRepoDeltaMap, size, backupTestRepoSizeMap},
						24316343,
					},
					"20210721-000101F",
					link,
					lsn,
					"",
					[]string{""},
					tablespace,
					struct {
						Start int64 "json:\"start\""
						Stop  int64 "json:\"stop\""
					}{1626825661, 1626825664},
					"full",
				},
				"y",
			},
			lastBackupsStruct{
				backupStruct{"20210721-000101F", "full", fullDate, 3, 24316343, 24316343, 2969514, backuptTestRepoDeltaMap, nil, backupTestRepoSizeMap, backupTestError, backupTestAnnotation, "y", []string{""}},
				backupStruct{"20210721-000101F", "diff", fullDate, 3, 24316343, 24316343, 2969514, backuptTestRepoDeltaMap, nil, backupTestRepoSizeMap, backupTestError, backupTestAnnotation, "y", []string{""}},
				backupStruct{"20210721-000101F", "incr", fullDate, 3, 24316343, 24316343, 2969514, backuptTestRepoDeltaMap, nil, backupTestRepoSizeMap, backupTestError, backupTestAnnotation, "y", []string{""}},
			},
		},
		{"compareLastBackupsDiff",
			args{
				&lastBackups,
				backup{
					nil,
					archive,
					backrestInfo,
					databaseID{1, 1},
					nil,
					backupTestError,
					backupInfo{
						2431634,
						struct {
							Delta    int64  "json:\"delta\""
							DeltaMap *int64 "json:\"delta-map\""
							Size     *int64 "json:\"size\""
							SizeMap  *int64 "json:\"size-map\""
						}{296951, backuptTestRepoDeltaMap, size, backupTestRepoSizeMap},
						2431634,
					},
					"20210721-000101F_20210721-000501D",
					link,
					lsn,
					"20210721-000101F",
					[]string{
						"20210721-000101F",
					},
					tablespace,
					struct {
						Start int64 "json:\"start\""
						Stop  int64 "json:\"stop\""
					}{1626825901, 1626825904},
					"diff",
				},
				"y",
			},
			lastBackupsStruct{
				backupStruct{"20210721-000101F", "full", fullDate, 3, 24316343, 24316343, 2969514, backuptTestRepoDeltaMap, nil, backupTestRepoSizeMap, backupTestError, backupTestAnnotation, "y", []string{""}},
				backupStruct{"20210721-000101F_20210721-000501D", "diff", diffDate, 3, 2431634, 2431634, 296951, backuptTestRepoDeltaMap, nil, backupTestRepoSizeMap, backupTestError, nil, "y", []string{"20210721-000101F"}},
				backupStruct{"20210721-000101F_20210721-000501D", "incr", diffDate, 3, 2431634, 2431634, 296951, backuptTestRepoDeltaMap, nil, backupTestRepoSizeMap, backupTestError, nil, "y", []string{"20210721-000101F"}},
			},
		},
		{"compareLastBackupsIncr",
			args{
				&lastBackups,
				backup{
					nil,
					archive,
					backrestInfo,
					databaseID{1, 1},
					nil,
					backupTestError,
					backupInfo{
						243163,
						struct {
							Delta    int64  "json:\"delta\""
							DeltaMap *int64 "json:\"delta-map\""
							Size     *int64 "json:\"size\""
							SizeMap  *int64 "json:\"size-map\""
						}{29695, backuptTestRepoDeltaMap, size, backupTestRepoSizeMap},
						243163,
					},
					"20210721-000101F_20210721-001001I",
					link,
					lsn,
					"20210721-000101F_20210721-000501D",
					[]string{
						"20210721-000101F",
						"20210721-000101F_20210721-000501D",
					},
					tablespace,
					struct {
						Start int64 "json:\"start\""
						Stop  int64 "json:\"stop\""
					}{1626826201, 1626826204},
					"incr",
				},
				"y",
			},
			lastBackupsStruct{
				backupStruct{"20210721-000101F", "full", fullDate, 3, 24316343, 24316343, 2969514, backuptTestRepoDeltaMap, nil, backupTestRepoSizeMap, backupTestError, backupTestAnnotation, "y", []string{""}},
				backupStruct{"20210721-000101F_20210721-000501D", "diff", diffDate, 3, 2431634, 2431634, 296951, backuptTestRepoDeltaMap, nil, backupTestRepoSizeMap, backupTestError, nil, "y", []string{"20210721-000101F"}},
				backupStruct{"20210721-000101F_20210721-001001I", "incr", incrDate, 3, 243163, 243163, 29695, backuptTestRepoDeltaMap, nil, backupTestRepoSizeMap, backupTestError, nil, "y", []string{"20210721-000101F", "20210721-000101F_20210721-000501D"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compareLastBackups(tt.args.backups, tt.args.backupTest, tt.args.backupIncr)
			if !reflect.DeepEqual(*tt.args.backups, tt.want) {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", *tt.args.backups, tt.want)
			}
		})
	}
}

func TestStanzaNotInExclude(t *testing.T) {
	type args struct {
		stanza      string
		listExclude []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"stanzaNotInExcludeEmptyListExclude",
			args{"", []string{""}},
			true},
		{"stanzaNotInExcludeEmptyListExcludeNotEmptyStanza",
			args{"demo", []string{""}},
			true},
		{"stanzaNotInExcludeStanzaNotInExcludeList",
			args{"demo", []string{"demo-test", "test"}},
			true},
		{"stanzaNotInExcludeStanzaInExcludeList",
			args{"demo", []string{"demo", "test"}},
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stanzaNotInExclude(tt.args.stanza, tt.args.listExclude); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func fakeSetUpMetricValue(metric *prometheus.GaugeVec, value float64, labels ...string) error {
	return errors.New("сustorm error for test")
}

//nolint:unparam
func templateStanza(walMax, walMin string, dbRef []databaseRef, errorStatus, stanzaLock bool, deltaMap, sizeMap, stanzaSizeTotal, stanzaSizeComplete int64, backupAnnotation annotation) stanza {
	var (
		size *int64
		link *[]struct {
			Destination string "json:\"destination\""
			Name        string "json:\"name\""
		}
		tablespace *[]struct {
			Destination string `json:"destination"`
			Name        string `json:"name"`
			OID         int    `json:"oid"`
		}
	)
	return stanza{
		[]archive{
			{databaseID{1, 1}, "13-1", walMax, walMin},
		},
		[]backup{
			{
				&backupAnnotation,
				struct {
					StartWAL string "json:\"start\""
					StopWAL  string "json:\"stop\""
				}{"000000010000000000000002", "000000010000000000000002"},
				struct {
					Format  int    "json:\"format\""
					Version string "json:\"version\""
				}{5, "2.45"},
				databaseID{1, 1},
				&dbRef,
				&errorStatus,
				backupInfo{
					24316343,
					struct {
						Delta    int64  "json:\"delta\""
						DeltaMap *int64 "json:\"delta-map\""
						Size     *int64 "json:\"size\""
						SizeMap  *int64 "json:\"size-map\""
					}{2969514, &deltaMap, size, &sizeMap},
					24316343,
				},
				"20210607-092423F",
				link,
				struct {
					StartLSN string "json:\"start\""
					StopLSN  string "json:\"stop\""
				}{"0/2000028", "0/2000100"},
				"",
				[]string{""},
				tablespace,
				struct {
					Start int64 "json:\"start\""
					Stop  int64 "json:\"stop\""
				}{1623057863, 1623057866},
				"full",
			},
		},
		"none",
		[]db{
			{1, 1, 6970977677138971135, "13"},
		},
		"demo",
		valToPtr([]repo{
			{"none",
				1,
				struct {
					Code    int    "json:\"code\""
					Message string "json:\"message\""
				}{0, "ok"},
			},
		}),
		status{
			0,
			struct {
				Backup struct {
					Held         bool   "json:\"held\""
					SizeTotal    *int64 `json:"size"`
					SizeComplete *int64 `json:"size-cplt"`
				} "json:\"backup\""
			}{
				struct {
					Held         bool   "json:\"held\""
					SizeTotal    *int64 `json:"size"`
					SizeComplete *int64 `json:"size-cplt"`
				}{stanzaLock, &stanzaSizeTotal, &stanzaSizeComplete},
			},
			"ok",
		},
	}
}

//nolint:unparam
func templateStanzaRepoMapSizesAbsent(walMax, walMin string, dbRef []databaseRef, errorStatus bool, size int64, backupAnnotation annotation) stanza {
	var (
		deltaMap, sizeMap, stanzaSizeTotal, stanzaSizeComplete *int64
		link                                                   *[]struct {
			Destination string "json:\"destination\""
			Name        string "json:\"name\""
		}
		tablespace *[]struct {
			Destination string `json:"destination"`
			Name        string `json:"name"`
			OID         int    `json:"oid"`
		}
	)
	return stanza{
		[]archive{
			{databaseID{1, 1}, "13-1", walMax, walMin},
		},
		[]backup{
			{
				&backupAnnotation,
				struct {
					StartWAL string "json:\"start\""
					StopWAL  string "json:\"stop\""
				}{"000000010000000000000002", "000000010000000000000002"},
				struct {
					Format  int    "json:\"format\""
					Version string "json:\"version\""
				}{5, "2.41"},
				databaseID{1, 1},
				&dbRef,
				&errorStatus,
				backupInfo{
					24316343,
					struct {
						Delta    int64  "json:\"delta\""
						DeltaMap *int64 "json:\"delta-map\""
						Size     *int64 "json:\"size\""
						SizeMap  *int64 "json:\"size-map\""
					}{2969514, deltaMap, &size, sizeMap},
					24316343,
				},
				"20210607-092423F",
				link,
				struct {
					StartLSN string "json:\"start\""
					StopLSN  string "json:\"stop\""
				}{"0/2000028", "0/2000100"},
				"",
				[]string{""},
				tablespace,
				struct {
					Start int64 "json:\"start\""
					Stop  int64 "json:\"stop\""
				}{1623057863, 1623057866},
				"full",
			},
		},
		"none",
		[]db{
			{1, 1, 6970977677138971135, "13"},
		},
		"demo",
		valToPtr([]repo{
			{"none",
				1,
				struct {
					Code    int    "json:\"code\""
					Message string "json:\"message\""
				}{0, "ok"},
			},
		}),
		status{
			0,
			struct {
				Backup struct {
					Held         bool   "json:\"held\""
					SizeTotal    *int64 `json:"size"`
					SizeComplete *int64 `json:"size-cplt"`
				} "json:\"backup\""
			}{
				struct {
					Held         bool   "json:\"held\""
					SizeTotal    *int64 `json:"size"`
					SizeComplete *int64 `json:"size-cplt"`
				}{false, stanzaSizeTotal, stanzaSizeComplete},
			},
			"ok",
		},
	}
}

//nolint:unparam
func templateStanzaDBsAbsent(walMax, walMin string, dbRef []databaseRef, errorStatus bool, size int64) stanza {
	var (
		deltaMap, sizeMap, stanzaSizeTotal, stanzaSizeComplete *int64
		link                                                   *[]struct {
			Destination string "json:\"destination\""
			Name        string "json:\"name\""
		}
		tablespace *[]struct {
			Destination string `json:"destination"`
			Name        string `json:"name"`
			OID         int    `json:"oid"`
		}
		backupTestAnnotation *annotation
	)
	return stanza{
		[]archive{
			{databaseID{1, 1}, "13-1", walMax, walMin},
		},
		[]backup{
			{
				backupTestAnnotation,
				struct {
					StartWAL string "json:\"start\""
					StopWAL  string "json:\"stop\""
				}{"000000010000000000000002", "000000010000000000000002"},
				struct {
					Format  int    "json:\"format\""
					Version string "json:\"version\""
				}{5, "2.41"},
				databaseID{1, 1},
				&dbRef,
				&errorStatus,
				backupInfo{
					24316343,
					struct {
						Delta    int64  "json:\"delta\""
						DeltaMap *int64 "json:\"delta-map\""
						Size     *int64 "json:\"size\""
						SizeMap  *int64 "json:\"size-map\""
					}{2969514, deltaMap, &size, sizeMap},
					24316343,
				},
				"20210607-092423F",
				link,
				struct {
					StartLSN string "json:\"start\""
					StopLSN  string "json:\"stop\""
				}{"0/2000028", "0/2000100"},
				"",
				[]string{""},
				tablespace,
				struct {
					Start int64 "json:\"start\""
					Stop  int64 "json:\"stop\""
				}{1623057863, 1623057866},
				"full",
			},
		},
		"none",
		[]db{
			{1, 1, 6970977677138971135, "13"},
		},
		"demo",
		valToPtr([]repo{
			{"none",
				1,
				struct {
					Code    int    "json:\"code\""
					Message string "json:\"message\""
				}{0, "ok"},
			},
		}),
		status{
			0,
			struct {
				Backup struct {
					Held         bool   "json:\"held\""
					SizeTotal    *int64 `json:"size"`
					SizeComplete *int64 `json:"size-cplt"`
				} "json:\"backup\""
			}{
				struct {
					Held         bool   "json:\"held\""
					SizeTotal    *int64 `json:"size"`
					SizeComplete *int64 `json:"size-cplt"`
				}{false, stanzaSizeTotal, stanzaSizeComplete},
			},
			"ok",
		},
	}
}

//nolint:unparam
func templateStanzaErrorAbsent(walMax, walMin string, size int64) stanza {
	var (
		errorStatus                                            *bool
		deltaMap, sizeMap, stanzaSizeTotal, stanzaSizeComplete *int64
		dbRef                                                  *[]databaseRef
		link                                                   *[]struct {
			Destination string "json:\"destination\""
			Name        string "json:\"name\""
		}
		tablespace *[]struct {
			Destination string `json:"destination"`
			Name        string `json:"name"`
			OID         int    `json:"oid"`
		}
	)
	return stanza{
		[]archive{
			{databaseID{1, 1}, "13-1", walMax, walMin},
		},
		[]backup{
			{
				nil,
				struct {
					StartWAL string "json:\"start\""
					StopWAL  string "json:\"stop\""
				}{"000000010000000000000002", "000000010000000000000002"},
				struct {
					Format  int    "json:\"format\""
					Version string "json:\"version\""
				}{5, "2.35"},
				databaseID{1, 1},
				dbRef,
				errorStatus,
				backupInfo{
					24316343,
					struct {
						Delta    int64  "json:\"delta\""
						DeltaMap *int64 "json:\"delta-map\""
						Size     *int64 "json:\"size\""
						SizeMap  *int64 "json:\"size-map\""
					}{2969514, deltaMap, &size, sizeMap},
					24316343,
				},
				"20210607-092423F",
				link,
				struct {
					StartLSN string "json:\"start\""
					StopLSN  string "json:\"stop\""
				}{"", ""},
				"",
				[]string{""},
				tablespace,
				struct {
					Start int64 "json:\"start\""
					Stop  int64 "json:\"stop\""
				}{1623057863, 1623057866},
				"full",
			},
		},
		"none",
		[]db{
			{1, 1, 6970977677138971135, "13"},
		},
		"demo",
		valToPtr([]repo{
			{"none",
				1,
				struct {
					Code    int    "json:\"code\""
					Message string "json:\"message\""
				}{0, "ok"},
			},
		}),
		status{
			0,
			struct {
				Backup struct {
					Held         bool   "json:\"held\""
					SizeTotal    *int64 `json:"size"`
					SizeComplete *int64 `json:"size-cplt"`
				} "json:\"backup\""
			}{
				struct {
					Held         bool   "json:\"held\""
					SizeTotal    *int64 `json:"size"`
					SizeComplete *int64 `json:"size-cplt"`
				}{false, stanzaSizeTotal, stanzaSizeComplete},
			},
			"ok",
		},
	}
}

//nolint:unparam
func templateStanzaRepoAbsent(walMax, walMin string, size int64) stanza {
	var (
		errorStatus                                            *bool
		deltaMap, sizeMap, stanzaSizeTotal, stanzaSizeComplete *int64
		dbRef                                                  *[]databaseRef
		repoInfo                                               *[]repo
		link                                                   *[]struct {
			Destination string "json:\"destination\""
			Name        string "json:\"name\""
		}
		tablespace *[]struct {
			Destination string `json:"destination"`
			Name        string `json:"name"`
			OID         int    `json:"oid"`
		}
		backupTestAnnotation *annotation
	)
	return stanza{
		[]archive{
			{databaseID{1, 0}, "13-1", walMax, walMin},
		},
		[]backup{
			{
				backupTestAnnotation,
				struct {
					StartWAL string "json:\"start\""
					StopWAL  string "json:\"stop\""
				}{"000000010000000000000002", "000000010000000000000002"},
				struct {
					Format  int    "json:\"format\""
					Version string "json:\"version\""
				}{5, "2.31"},
				databaseID{1, 0},
				dbRef,
				errorStatus,
				backupInfo{
					24316343,
					struct {
						Delta    int64  "json:\"delta\""
						DeltaMap *int64 "json:\"delta-map\""
						Size     *int64 "json:\"size\""
						SizeMap  *int64 "json:\"size-map\""
					}{2969514, deltaMap, &size, sizeMap},
					24316343,
				},
				"20210607-092423F",
				link,
				struct {
					StartLSN string "json:\"start\""
					StopLSN  string "json:\"stop\""
				}{"", ""},
				"",
				[]string{""},
				tablespace,
				struct {
					Start int64 "json:\"start\""
					Stop  int64 "json:\"stop\""
				}{1623057863, 1623057866},
				"full",
			},
		},
		"none",
		[]db{
			{1, 0, 6970977677138971135, "13"},
		},
		"demo",
		repoInfo,
		status{
			0,
			struct {
				Backup struct {
					Held         bool   "json:\"held\""
					SizeTotal    *int64 `json:"size"`
					SizeComplete *int64 `json:"size-cplt"`
				} "json:\"backup\""
			}{
				struct {
					Held         bool   "json:\"held\""
					SizeTotal    *int64 `json:"size"`
					SizeComplete *int64 `json:"size-cplt"`
				}{false, stanzaSizeTotal, stanzaSizeComplete},
			},
			"ok",
		},
	}
}

func parseDate(value string) time.Time {
	loc, err := time.LoadLocation("Local")
	if err != nil {
		panic(err)
	}
	valueReturn, err := time.ParseInLocation(layout, value, loc)
	if err != nil {
		panic(err)
	}
	return valueReturn
}

func TestGetParsedSpecificBackupInfoDataErrors(t *testing.T) {
	type args struct {
		config            string
		configIncludePath string
		stanzaName        string
		backupLabel       string
		errorsCount       int
	}
	tests := []struct {
		name         string
		args         args
		mockTestData mockStruct
	}{
		{
			"getParsedSpecificBackupInfoDataErrors",
			args{
				"",
				"",
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
					annotation{"testkey": "testvalue"}).Backup[0].Label,
				3,
			},
			// Imitate error, when pgBackRest binary not found.
			mockStruct{
				``,
				`executable file not found in $PATH`,
				127,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockData = tt.mockTestData
			execCommand = fakeExecCommand
			defer func() { execCommand = exec.Command }()
			out := &bytes.Buffer{}
			lc := log.NewLogfmtLogger(out)
			getParsedSpecificBackupInfoData(tt.args.config, tt.args.configIncludePath, tt.args.stanzaName, tt.args.backupLabel, lc)
			errorsOutputCount := strings.Count(out.String(), "level=error")
			if tt.args.errorsCount != errorsOutputCount {
				t.Errorf("\nVariables do not match:\nerrors=%d, want:\nerrors=%d",
					tt.args.errorsCount, errorsOutputCount)
			}
		})
	}
}

func fakeExecCommandSpecificDatabase(command string, args ...string) *exec.Cmd {
	var (
		stdOut, stdErr string
		ecode          int
	)
	cs := []string{"-test.run=TestExecCommandHelper", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	switch {
	case checkBackupType(cs, `D$`):
		stdOut = mockDataBackupLast.mockDiff.mockStdout
		stdErr = mockDataBackupLast.mockDiff.mockStderr
		ecode = mockDataBackupLast.mockDiff.mockExit
	case checkBackupType(cs, `I$`):
		stdOut = mockDataBackupLast.mockIncr.mockStdout
		stdErr = mockDataBackupLast.mockIncr.mockStderr
		ecode = mockDataBackupLast.mockIncr.mockExit
	default:
		stdOut = mockDataBackupLast.mockFull.mockStdout
		stdErr = mockDataBackupLast.mockFull.mockStderr
		ecode = mockDataBackupLast.mockFull.mockExit
	}
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1",
		"STDOUT=" + stdOut,
		"STDERR=" + stdErr,
		"EXIT_STATUS=" + strconv.Itoa(ecode)}
	return cmd
}

func checkBackupType(a []string, regex string) bool {
	for _, n := range a {
		found, err := regexp.MatchString(regex, n)
		if err != nil {
			panic(err)
		}
		if found {
			return true
		}
	}
	return false
}

//nolint:unparam
func templateLastBackup() lastBackupsStruct {
	return lastBackupsStruct{
		backupStruct{"20210607-092423F", "full", time.Unix(1623057866, 0), 3, 24316343, 24316343, 2969514, valToPtr(int64(12)), nil, valToPtr(int64(100)), valToPtr(true), valToPtr(annotation{"testkey": "testvalue"}), "y", []string{""}},
		backupStruct{"20210607-092423F", "diff", time.Unix(1623057866, 0), 3, 24316343, 24316343, 2969514, valToPtr(int64(12)), nil, valToPtr(int64(100)), valToPtr(true), valToPtr(annotation{"testkey": "testvalue"}), "y", []string{""}},
		backupStruct{"20210607-092423F", "incr", time.Unix(1623057866, 0), 3, 24316343, 24316343, 2969514, valToPtr(int64(12)), nil, valToPtr(int64(100)), valToPtr(true), valToPtr(annotation{"testkey": "testvalue"}), "y", []string{""}},
	}
}

func templateLastBackupRepoMapSizesAbsent() lastBackupsStruct {
	return lastBackupsStruct{
		backupStruct{"20210607-092423F", "full", time.Unix(1623057866, 0), 3, 24316343, 24316343, 2969514, nil, valToPtr(int64(2969514)), nil, valToPtr(true), valToPtr(annotation{"testkey": "testvalue"}), "n", []string{""}},
		backupStruct{"20210607-092423F", "diff", time.Unix(1623057866, 0), 3, 24316343, 24316343, 2969514, nil, valToPtr(int64(2969514)), nil, valToPtr(true), valToPtr(annotation{"testkey": "testvalue"}), "n", []string{""}},
		backupStruct{"20210607-092423F", "incr", time.Unix(1623057866, 0), 3, 24316343, 24316343, 2969514, nil, valToPtr(int64(2969514)), nil, valToPtr(true), valToPtr(annotation{"testkey": "testvalue"}), "n", []string{""}},
	}
}

func templateLastBackupDBsAbsent() lastBackupsStruct {
	return lastBackupsStruct{
		backupStruct{"20210607-092423F", "full", time.Unix(1623057866, 0), 3, 24316343, 24316343, 2969514, nil, valToPtr(int64(2969514)), nil, valToPtr(true), nil, "n", []string{""}},
		backupStruct{"20210607-092423F", "diff", time.Unix(1623057866, 0), 3, 24316343, 24316343, 2969514, nil, valToPtr(int64(2969514)), nil, valToPtr(true), nil, "n", []string{""}},
		backupStruct{"20210607-092423F", "incr", time.Unix(1623057866, 0), 3, 24316343, 24316343, 2969514, nil, valToPtr(int64(2969514)), nil, valToPtr(true), nil, "n", []string{""}},
	}
}

func templateLastBackupErrorAbsent() lastBackupsStruct {
	return lastBackupsStruct{
		backupStruct{"20210607-092423F", "full", time.Unix(1623057866, 0), 3, 24316343, 24316343, 2969514, nil, valToPtr(int64(2969514)), nil, nil, nil, "n", []string{""}},
		backupStruct{"20210607-092423F", "diff", time.Unix(1623057866, 0), 3, 24316343, 24316343, 2969514, nil, valToPtr(int64(2969514)), nil, nil, nil, "n", []string{""}},
		backupStruct{"20210607-092423F", "incr", time.Unix(1623057866, 0), 3, 24316343, 24316343, 2969514, nil, valToPtr(int64(2969514)), nil, nil, nil, "n", []string{""}},
	}
}

//nolint:unparam
func templateLastBackupDifferent() lastBackupsStruct {
	return lastBackupsStruct{
		backupStruct{"20220926-201857F", "full", time.Unix(1623706322, 0), 3, 24316343, 24316343, 2969514, valToPtr(int64(12)), nil, valToPtr(int64(100)), valToPtr(false), valToPtr(annotation{"testkey": "testvalue"}), "y", []string{""}},
		backupStruct{"20220926-201857F_20220926-201901D", "diff", time.Unix(1623706322, 0), 3, 32230330, 32230330, 2969514, valToPtr(int64(12)), nil, valToPtr(int64(100)), valToPtr(false), nil, "y", []string{"20220926-201857F"}},
		backupStruct{"20220926-201857F_20220926-202454I", "incr", time.Unix(1623706322, 0), 3, 32230330, 32230330, 2969514, valToPtr(int64(12)), nil, valToPtr(int64(100)), valToPtr(false), nil, "y", []string{"20220926-201857F", "20220926-201857F_20220926-201901D"}},
	}
}

// Implement custom comparators for testing.
//
//nolint:gocyclo
func compareBackupStructs(a, b backupStruct) bool {
	if a.backupLabel != b.backupLabel {
		return false
	}
	if a.backupType != b.backupType {
		return false
	}
	if !a.backupTime.Equal(b.backupTime) {
		return false
	}
	if a.backupDuration != b.backupDuration {
		return false
	}
	if a.backupDelta != b.backupDelta {
		return false
	}
	if a.backupSize != b.backupSize {
		return false
	}
	if a.backupRepoDelta != b.backupRepoDelta {
		return false
	}
	if a.backupRepoDeltaMap != nil && b.backupRepoDeltaMap != nil {
		if *a.backupRepoDeltaMap != *b.backupRepoDeltaMap {
			return false
		}
	} else if a.backupRepoDeltaMap != b.backupRepoDeltaMap {
		return false
	}
	if a.backupRepoSize != nil && b.backupRepoSize != nil {
		if *a.backupRepoSize != *b.backupRepoSize {
			return false
		}
	} else if a.backupRepoSize != b.backupRepoSize {
		return false
	}
	if a.backupRepoSizeMap != nil && b.backupRepoSizeMap != nil {
		if *a.backupRepoSizeMap != *b.backupRepoSizeMap {
			return false
		}
	} else if a.backupRepoSizeMap != b.backupRepoSizeMap {
		return false
	}
	if a.backupError != nil && b.backupError != nil {
		if *a.backupError != *b.backupError {
			return false
		}
	} else if a.backupError != b.backupError {
		return false
	}
	if a.backupAnnotation != nil && b.backupAnnotation != nil {
		if !reflect.DeepEqual(*a.backupAnnotation, *b.backupAnnotation) {
			return false
		}
	} else if a.backupAnnotation != b.backupAnnotation {
		return false
	}
	if a.backupBlockIncr != b.backupBlockIncr {
		return false
	}
	return true
}

func mapsEqual(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

func printSortedMap(name string, m map[string]int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Printf("%s:\n", name)
	for _, k := range keys {
		fmt.Printf("  %s: %d\n", k, m[k])
	}
}

func TestGetBackupReferencesTotal(t *testing.T) {
	tests := []struct {
		name    string
		refList []string
		want    map[string]int
		wantErr bool
	}{
		{
			name:    "Empty reference list",
			refList: []string{},
			want:    map[string]int{fullLabel: 0, diffLabel: 0, incrLabel: 0},
			wantErr: false,
		},
		{
			name:    "Valid references 1",
			refList: []string{"20220926-201857F", "20220926-201857F_20220926-201901D"},
			want:    map[string]int{fullLabel: 1, diffLabel: 1, incrLabel: 0},
			wantErr: false,
		},
		{
			name:    "Valid references 2",
			refList: []string{"20220926-201857F", "20220926-201857F_20220926-201901D", "20220926-201857F_20220926-202454I"},
			want:    map[string]int{fullLabel: 1, diffLabel: 1, incrLabel: 1},
			wantErr: false,
		},
		{
			name:    "Invalid reference",
			refList: []string{"20220926-201857F", "invalid_backup"},
			want:    map[string]int{fullLabel: 1, diffLabel: 0, incrLabel: 0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getBackupReferencesTotal(tt.refList)
			if (err != nil) != tt.wantErr {
				t.Errorf("\nVariables do not match:\n%v\nwantErr:\n%v", err, tt.wantErr)
				return
			}
			if !maps.Equal(got, tt.want) {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}
