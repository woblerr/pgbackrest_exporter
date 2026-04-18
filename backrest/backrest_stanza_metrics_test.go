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
func TestGetStanzaMetrics(t *testing.T) {
	type args struct {
		stanzaName          string
		stanzaStatus        status
		stanzaRepo          *[]repo
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
			"getStanzaMetricsBackupInProgress",
			args{
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					true,
					false,
					12,
					100,
					12345,
					1234,
					0,
					0,
					annotation{"testkey": "testvalue"}).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					true,
					false,
					12,
					100,
					12345,
					1234,
					0,
					0,
					annotation{"testkey": "testvalue"}).Status,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					true,
					false,
					12,
					100,
					12345,
					1234,
					0,
					0,
					annotation{"testkey": "testvalue"}).Repo,
				setUpMetricValue,
				`# HELP pgbackrest_stanza_backup_complete_bytes Completed size for backup in progress.
# TYPE pgbackrest_stanza_backup_complete_bytes gauge
pgbackrest_stanza_backup_complete_bytes{stanza="demo"} 1234
# HELP pgbackrest_stanza_backup_lock_status Current stanza backup lock status.
# TYPE pgbackrest_stanza_backup_lock_status gauge
pgbackrest_stanza_backup_lock_status{stanza="demo"} 1
# HELP pgbackrest_stanza_backup_repo_complete_bytes Completed size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_complete_bytes gauge
pgbackrest_stanza_backup_repo_complete_bytes{repo_key="1",stanza="demo"} 0
# HELP pgbackrest_stanza_backup_repo_total_bytes Total size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_total_bytes gauge
pgbackrest_stanza_backup_repo_total_bytes{repo_key="1",stanza="demo"} 0
# HELP pgbackrest_stanza_backup_total_bytes Total size for backup in progress.
# TYPE pgbackrest_stanza_backup_total_bytes gauge
pgbackrest_stanza_backup_total_bytes{stanza="demo"} 12345
# HELP pgbackrest_stanza_restore_complete_bytes Completed size for restore in progress.
# TYPE pgbackrest_stanza_restore_complete_bytes gauge
pgbackrest_stanza_restore_complete_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_lock_status Current stanza restore lock status.
# TYPE pgbackrest_stanza_restore_lock_status gauge
pgbackrest_stanza_restore_lock_status{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_total_bytes Total size for restore in progress.
# TYPE pgbackrest_stanza_restore_total_bytes gauge
pgbackrest_stanza_restore_total_bytes{stanza="demo"} 0
` + templateMetrics,
			},
		},
		{
			"getStanzaMetricsBackupNotInProgress",
			args{
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					false,
					false,
					12,
					100,
					0,
					0,
					0,
					0,
					annotation{"testkey": "testvalue"}).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					false,
					false,
					12,
					100,
					0,
					0,
					0,
					0,
					annotation{"testkey": "testvalue"}).Status,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					false,
					false,
					12,
					100,
					0,
					0,
					0,
					0,
					annotation{"testkey": "testvalue"}).Repo,
				setUpMetricValue,
				`# HELP pgbackrest_stanza_backup_complete_bytes Completed size for backup in progress.
# TYPE pgbackrest_stanza_backup_complete_bytes gauge
pgbackrest_stanza_backup_complete_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_backup_lock_status Current stanza backup lock status.
# TYPE pgbackrest_stanza_backup_lock_status gauge
pgbackrest_stanza_backup_lock_status{stanza="demo"} 0
# HELP pgbackrest_stanza_backup_repo_complete_bytes Completed size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_complete_bytes gauge
pgbackrest_stanza_backup_repo_complete_bytes{repo_key="1",stanza="demo"} 0
# HELP pgbackrest_stanza_backup_repo_total_bytes Total size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_total_bytes gauge
pgbackrest_stanza_backup_repo_total_bytes{repo_key="1",stanza="demo"} 0
# HELP pgbackrest_stanza_backup_total_bytes Total size for backup in progress.
# TYPE pgbackrest_stanza_backup_total_bytes gauge
pgbackrest_stanza_backup_total_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_complete_bytes Completed size for restore in progress.
# TYPE pgbackrest_stanza_restore_complete_bytes gauge
pgbackrest_stanza_restore_complete_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_lock_status Current stanza restore lock status.
# TYPE pgbackrest_stanza_restore_lock_status gauge
pgbackrest_stanza_restore_lock_status{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_total_bytes Total size for restore in progress.
# TYPE pgbackrest_stanza_restore_total_bytes gauge
pgbackrest_stanza_restore_total_bytes{stanza="demo"} 0
` + templateMetrics,
			},
		},
		{
			"getStanzaMetricsRestoreInProgress",
			args{
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					false,
					true,
					12,
					100,
					0,
					0,
					12345,
					1234,
					annotation{"testkey": "testvalue"}).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					false,
					true,
					12,
					100,
					0,
					0,
					12345,
					1234,
					annotation{"testkey": "testvalue"}).Status,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					false,
					true,
					12,
					100,
					0,
					0,
					12345,
					1234,
					annotation{"testkey": "testvalue"}).Repo,
				setUpMetricValue,
				`# HELP pgbackrest_stanza_backup_complete_bytes Completed size for backup in progress.
# TYPE pgbackrest_stanza_backup_complete_bytes gauge
pgbackrest_stanza_backup_complete_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_backup_lock_status Current stanza backup lock status.
# TYPE pgbackrest_stanza_backup_lock_status gauge
pgbackrest_stanza_backup_lock_status{stanza="demo"} 0
# HELP pgbackrest_stanza_backup_repo_complete_bytes Completed size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_complete_bytes gauge
pgbackrest_stanza_backup_repo_complete_bytes{repo_key="1",stanza="demo"} 0
# HELP pgbackrest_stanza_backup_repo_total_bytes Total size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_total_bytes gauge
pgbackrest_stanza_backup_repo_total_bytes{repo_key="1",stanza="demo"} 0
# HELP pgbackrest_stanza_backup_total_bytes Total size for backup in progress.
# TYPE pgbackrest_stanza_backup_total_bytes gauge
pgbackrest_stanza_backup_total_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_complete_bytes Completed size for restore in progress.
# TYPE pgbackrest_stanza_restore_complete_bytes gauge
pgbackrest_stanza_restore_complete_bytes{stanza="demo"} 1234
# HELP pgbackrest_stanza_restore_lock_status Current stanza restore lock status.
# TYPE pgbackrest_stanza_restore_lock_status gauge
pgbackrest_stanza_restore_lock_status{stanza="demo"} 1
# HELP pgbackrest_stanza_restore_total_bytes Total size for restore in progress.
# TYPE pgbackrest_stanza_restore_total_bytes gauge
pgbackrest_stanza_restore_total_bytes{stanza="demo"} 12345
` + templateMetrics,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetMetrics()
			getStanzaMetrics(tt.args.stanzaName, tt.args.stanzaStatus, tt.args.stanzaRepo, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaStatusMetric,
				pgbrStanzaBackupLockStatusMetric,
				pgbrStanzaBackupInProgressTotalMetric,
				pgbrStanzaBackupInProgressCompleteMetric,
				pgbrStanzaBackupInProgressRepoTotalMetric,
				pgbrStanzaBackupInProgressRepoCompleteMetric,
				pgbrStanzaRestoreLockStatusMetric,
				pgbrStanzaRestoreInProgressTotalMetric,
				pgbrStanzaRestoreInProgressCompleteMetric,
			)
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

// pgBackrest version >= 2.59.
// Per-repo backup progress metrics have real values from Lock.Backup.Repo:
//   - pgbackrest_stanza_backup_repo_total_bytes
//   - pgbackrest_stanza_backup_repo_complete_bytes
func TestGetStanzaMetricsBackupRepoProgress(t *testing.T) {
	type args struct {
		stanzaName          string
		stanzaStatus        status
		stanzaRepo          *[]repo
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
	}
	lockBackupRepos := []lockBackupRepo{
		{Key: 1, SizeTotal: 3159000, SizeComplete: 1754830},
		{Key: 2, SizeTotal: 3159000, SizeComplete: 2369250},
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"getStanzaMetricsBackupRepoProgressSingleRepo",
			args{
				templateStanzaBackupRepoProgress(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100,
					6318000,
					4124080,
					annotation{"testkey": "testvalue"},
					lockBackupRepos[:1]).Name,
				templateStanzaBackupRepoProgress(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100,
					6318000,
					4124080,
					annotation{"testkey": "testvalue"},
					lockBackupRepos[:1]).Status,
				templateStanzaBackupRepoProgress(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100,
					6318000,
					4124080,
					annotation{"testkey": "testvalue"},
					lockBackupRepos[:1]).Repo,
				setUpMetricValue,
				`# HELP pgbackrest_stanza_backup_complete_bytes Completed size for backup in progress.
# TYPE pgbackrest_stanza_backup_complete_bytes gauge
pgbackrest_stanza_backup_complete_bytes{stanza="demo"} 4.12408e+06
# HELP pgbackrest_stanza_backup_lock_status Current stanza backup lock status.
# TYPE pgbackrest_stanza_backup_lock_status gauge
pgbackrest_stanza_backup_lock_status{stanza="demo"} 1
# HELP pgbackrest_stanza_backup_repo_complete_bytes Completed size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_complete_bytes gauge
pgbackrest_stanza_backup_repo_complete_bytes{repo_key="1",stanza="demo"} 1.75483e+06
# HELP pgbackrest_stanza_backup_repo_total_bytes Total size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_total_bytes gauge
pgbackrest_stanza_backup_repo_total_bytes{repo_key="1",stanza="demo"} 3.159e+06
# HELP pgbackrest_stanza_backup_total_bytes Total size for backup in progress.
# TYPE pgbackrest_stanza_backup_total_bytes gauge
pgbackrest_stanza_backup_total_bytes{stanza="demo"} 6.318e+06
# HELP pgbackrest_stanza_restore_complete_bytes Completed size for restore in progress.
# TYPE pgbackrest_stanza_restore_complete_bytes gauge
pgbackrest_stanza_restore_complete_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_lock_status Current stanza restore lock status.
# TYPE pgbackrest_stanza_restore_lock_status gauge
pgbackrest_stanza_restore_lock_status{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_total_bytes Total size for restore in progress.
# TYPE pgbackrest_stanza_restore_total_bytes gauge
pgbackrest_stanza_restore_total_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_status Current stanza status.
# TYPE pgbackrest_stanza_status gauge
pgbackrest_stanza_status{stanza="demo"} 0
`,
			},
		},
		{
			"getStanzaMetricsBackupRepoProgressMultiRepo",
			args{
				templateStanzaBackupRepoProgress(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100,
					6318000,
					4124080,
					annotation{"testkey": "testvalue"},
					lockBackupRepos).Name,
				templateStanzaBackupRepoProgress(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100,
					6318000,
					4124080,
					annotation{"testkey": "testvalue"},
					lockBackupRepos).Status,
				templateStanzaBackupRepoProgress(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					12,
					100,
					6318000,
					4124080,
					annotation{"testkey": "testvalue"},
					lockBackupRepos).Repo,
				setUpMetricValue,
				`# HELP pgbackrest_stanza_backup_complete_bytes Completed size for backup in progress.
# TYPE pgbackrest_stanza_backup_complete_bytes gauge
pgbackrest_stanza_backup_complete_bytes{stanza="demo"} 4.12408e+06
# HELP pgbackrest_stanza_backup_lock_status Current stanza backup lock status.
# TYPE pgbackrest_stanza_backup_lock_status gauge
pgbackrest_stanza_backup_lock_status{stanza="demo"} 1
# HELP pgbackrest_stanza_backup_repo_complete_bytes Completed size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_complete_bytes gauge
pgbackrest_stanza_backup_repo_complete_bytes{repo_key="1",stanza="demo"} 1.75483e+06
pgbackrest_stanza_backup_repo_complete_bytes{repo_key="2",stanza="demo"} 2.36925e+06
# HELP pgbackrest_stanza_backup_repo_total_bytes Total size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_total_bytes gauge
pgbackrest_stanza_backup_repo_total_bytes{repo_key="1",stanza="demo"} 3.159e+06
pgbackrest_stanza_backup_repo_total_bytes{repo_key="2",stanza="demo"} 3.159e+06
# HELP pgbackrest_stanza_backup_total_bytes Total size for backup in progress.
# TYPE pgbackrest_stanza_backup_total_bytes gauge
pgbackrest_stanza_backup_total_bytes{stanza="demo"} 6.318e+06
# HELP pgbackrest_stanza_restore_complete_bytes Completed size for restore in progress.
# TYPE pgbackrest_stanza_restore_complete_bytes gauge
pgbackrest_stanza_restore_complete_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_lock_status Current stanza restore lock status.
# TYPE pgbackrest_stanza_restore_lock_status gauge
pgbackrest_stanza_restore_lock_status{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_total_bytes Total size for restore in progress.
# TYPE pgbackrest_stanza_restore_total_bytes gauge
pgbackrest_stanza_restore_total_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_status Current stanza status.
# TYPE pgbackrest_stanza_status gauge
pgbackrest_stanza_status{stanza="demo"} 0
`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetMetrics()
			getStanzaMetrics(tt.args.stanzaName, tt.args.stanzaStatus, tt.args.stanzaRepo, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaStatusMetric,
				pgbrStanzaBackupLockStatusMetric,
				pgbrStanzaBackupInProgressTotalMetric,
				pgbrStanzaBackupInProgressCompleteMetric,
				pgbrStanzaBackupInProgressRepoTotalMetric,
				pgbrStanzaBackupInProgressRepoCompleteMetric,
				pgbrStanzaRestoreLockStatusMetric,
				pgbrStanzaRestoreInProgressTotalMetric,
				pgbrStanzaRestoreInProgressCompleteMetric,
			)
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

// pgBackrest version < 2.56.
// Metrics always have 0 value:
//   - pgbackrest_stanza_restore_lock_status
//   - pgbackrest_stanza_restore_total_bytes
//   - pgbackrest_stanza_restore_complete_bytes
func TestGetStanzaMetricsRestoreProgressAbsent(t *testing.T) {
	type args struct {
		stanzaName          string
		stanzaStatus        status
		stanzaRepo          *[]repo
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
			"getStanzaMetricsBackupInProgress",
			args{
				templateStanzaRestoreLockAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					true,
					12,
					100,
					12345,
					1234,
					annotation{"testkey": "testvalue"}).Name,
				templateStanzaRestoreLockAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					true,
					12,
					100,
					12345,
					1234,
					annotation{"testkey": "testvalue"}).Status,
				templateStanzaRestoreLockAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					true,
					12,
					100,
					12345,
					1234,
					annotation{"testkey": "testvalue"}).Repo,
				setUpMetricValue,
				`# HELP pgbackrest_stanza_backup_complete_bytes Completed size for backup in progress.
# TYPE pgbackrest_stanza_backup_complete_bytes gauge
pgbackrest_stanza_backup_complete_bytes{stanza="demo"} 1234
# HELP pgbackrest_stanza_backup_lock_status Current stanza backup lock status.
# TYPE pgbackrest_stanza_backup_lock_status gauge
pgbackrest_stanza_backup_lock_status{stanza="demo"} 1
# HELP pgbackrest_stanza_backup_repo_complete_bytes Completed size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_complete_bytes gauge
pgbackrest_stanza_backup_repo_complete_bytes{repo_key="1",stanza="demo"} 0
# HELP pgbackrest_stanza_backup_repo_total_bytes Total size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_total_bytes gauge
pgbackrest_stanza_backup_repo_total_bytes{repo_key="1",stanza="demo"} 0
# HELP pgbackrest_stanza_backup_total_bytes Total size for backup in progress.
# TYPE pgbackrest_stanza_backup_total_bytes gauge
pgbackrest_stanza_backup_total_bytes{stanza="demo"} 12345
# HELP pgbackrest_stanza_restore_complete_bytes Completed size for restore in progress.
# TYPE pgbackrest_stanza_restore_complete_bytes gauge
pgbackrest_stanza_restore_complete_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_lock_status Current stanza restore lock status.
# TYPE pgbackrest_stanza_restore_lock_status gauge
pgbackrest_stanza_restore_lock_status{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_total_bytes Total size for restore in progress.
# TYPE pgbackrest_stanza_restore_total_bytes gauge
pgbackrest_stanza_restore_total_bytes{stanza="demo"} 0
` + templateMetrics,
			},
		},
		{
			"getStanzaMetricsBackupNotInProgress",
			args{
				templateStanzaRestoreLockAbsent(
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
				templateStanzaRestoreLockAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					false,
					12,
					100,
					0,
					0,
					annotation{"testkey": "testvalue"}).Status,
				templateStanzaRestoreLockAbsent(
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
				`# HELP pgbackrest_stanza_backup_complete_bytes Completed size for backup in progress.
# TYPE pgbackrest_stanza_backup_complete_bytes gauge
pgbackrest_stanza_backup_complete_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_backup_lock_status Current stanza backup lock status.
# TYPE pgbackrest_stanza_backup_lock_status gauge
pgbackrest_stanza_backup_lock_status{stanza="demo"} 0
# HELP pgbackrest_stanza_backup_repo_complete_bytes Completed size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_complete_bytes gauge
pgbackrest_stanza_backup_repo_complete_bytes{repo_key="1",stanza="demo"} 0
# HELP pgbackrest_stanza_backup_repo_total_bytes Total size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_total_bytes gauge
pgbackrest_stanza_backup_repo_total_bytes{repo_key="1",stanza="demo"} 0
# HELP pgbackrest_stanza_backup_total_bytes Total size for backup in progress.
# TYPE pgbackrest_stanza_backup_total_bytes gauge
pgbackrest_stanza_backup_total_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_complete_bytes Completed size for restore in progress.
# TYPE pgbackrest_stanza_restore_complete_bytes gauge
pgbackrest_stanza_restore_complete_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_lock_status Current stanza restore lock status.
# TYPE pgbackrest_stanza_restore_lock_status gauge
pgbackrest_stanza_restore_lock_status{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_total_bytes Total size for restore in progress.
# TYPE pgbackrest_stanza_restore_total_bytes gauge
pgbackrest_stanza_restore_total_bytes{stanza="demo"} 0
` + templateMetrics,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetMetrics()
			getStanzaMetrics(tt.args.stanzaName, tt.args.stanzaStatus, tt.args.stanzaRepo, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaStatusMetric,
				pgbrStanzaBackupLockStatusMetric,
				pgbrStanzaBackupInProgressTotalMetric,
				pgbrStanzaBackupInProgressCompleteMetric,
				pgbrStanzaBackupInProgressRepoTotalMetric,
				pgbrStanzaBackupInProgressRepoCompleteMetric,
				pgbrStanzaRestoreLockStatusMetric,
				pgbrStanzaRestoreInProgressTotalMetric,
				pgbrStanzaRestoreInProgressCompleteMetric,
			)
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

// pgBackrest version < 2.48.
// Metrics always have 0 value:
//   - pgbackrest_stanza_backup_total_bytes
//   - pgbackrest_stanza_backup_complete_bytes
//   - pgbackrest_stanza_backup_repo_complete_bytes
//   - pgbackrest_stanza_backup_repo_total_bytes
//   - pgbackrest_stanza_restore_lock_status
//   - pgbackrest_stanza_restore_total_bytes
//   - pgbackrest_stanza_restore_complete_bytes
func TestGetStanzaMetricsBackupProgressAbsent(t *testing.T) {
	type args struct {
		stanzaName          string
		stanzaStatus        status
		stanzaRepo          *[]repo
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
	}
	templateMetrics := `# HELP pgbackrest_stanza_backup_complete_bytes Completed size for backup in progress.
# TYPE pgbackrest_stanza_backup_complete_bytes gauge
pgbackrest_stanza_backup_complete_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_backup_lock_status Current stanza backup lock status.
# TYPE pgbackrest_stanza_backup_lock_status gauge
pgbackrest_stanza_backup_lock_status{stanza="demo"} 0
# HELP pgbackrest_stanza_backup_repo_complete_bytes Completed size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_complete_bytes gauge
pgbackrest_stanza_backup_repo_complete_bytes{repo_key="1",stanza="demo"} 0
# HELP pgbackrest_stanza_backup_repo_total_bytes Total size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_total_bytes gauge
pgbackrest_stanza_backup_repo_total_bytes{repo_key="1",stanza="demo"} 0
# HELP pgbackrest_stanza_backup_total_bytes Total size for backup in progress.
# TYPE pgbackrest_stanza_backup_total_bytes gauge
pgbackrest_stanza_backup_total_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_complete_bytes Completed size for restore in progress.
# TYPE pgbackrest_stanza_restore_complete_bytes gauge
pgbackrest_stanza_restore_complete_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_lock_status Current stanza restore lock status.
# TYPE pgbackrest_stanza_restore_lock_status gauge
pgbackrest_stanza_restore_lock_status{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_total_bytes Total size for restore in progress.
# TYPE pgbackrest_stanza_restore_total_bytes gauge
pgbackrest_stanza_restore_total_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_status Current stanza status.
# TYPE pgbackrest_stanza_status gauge
pgbackrest_stanza_status{stanza="demo"} 0
`
	tests := []struct {
		name string
		args args
	}{
		{
			"getStanzaMetricsBackupInProgress",
			args{
				templateStanzaRepoMapSizesAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					2969514,
					annotation{"testkey": "testvalue"}).Name,
				templateStanzaRepoMapSizesAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					2969514,
					annotation{"testkey": "testvalue"}).Status,
				templateStanzaRepoMapSizesAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					2969514,
					annotation{"testkey": "testvalue"}).Repo,
				setUpMetricValue,
				templateMetrics,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetMetrics()
			getStanzaMetrics(tt.args.stanzaName, tt.args.stanzaStatus, tt.args.stanzaRepo, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaStatusMetric,
				pgbrStanzaBackupLockStatusMetric,
				pgbrStanzaBackupInProgressTotalMetric,
				pgbrStanzaBackupInProgressCompleteMetric,
				pgbrStanzaBackupInProgressRepoTotalMetric,
				pgbrStanzaBackupInProgressRepoCompleteMetric,
				pgbrStanzaRestoreLockStatusMetric,
				pgbrStanzaRestoreInProgressTotalMetric,
				pgbrStanzaRestoreInProgressCompleteMetric,
			)
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

// pgBackrest version < 2.32.
// Stanza repo list is absent, per-repo backup progress metrics use repo_key="0" with value 0:
//   - pgbackrest_stanza_backup_repo_total_bytes
//   - pgbackrest_stanza_backup_repo_complete_bytes
func TestGetStanzaMetricsStanzaRepoAbsent(t *testing.T) {
	type args struct {
		stanzaName          string
		stanzaStatus        status
		stanzaRepo          *[]repo
		setUpMetricValueFun setUpMetricValueFunType
		testText            string
	}
	templateMetrics := `# HELP pgbackrest_stanza_backup_complete_bytes Completed size for backup in progress.
# TYPE pgbackrest_stanza_backup_complete_bytes gauge
pgbackrest_stanza_backup_complete_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_backup_lock_status Current stanza backup lock status.
# TYPE pgbackrest_stanza_backup_lock_status gauge
pgbackrest_stanza_backup_lock_status{stanza="demo"} 0
# HELP pgbackrest_stanza_backup_repo_complete_bytes Completed size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_complete_bytes gauge
pgbackrest_stanza_backup_repo_complete_bytes{repo_key="0",stanza="demo"} 0
# HELP pgbackrest_stanza_backup_repo_total_bytes Total size for backup in progress per repository.
# TYPE pgbackrest_stanza_backup_repo_total_bytes gauge
pgbackrest_stanza_backup_repo_total_bytes{repo_key="0",stanza="demo"} 0
# HELP pgbackrest_stanza_backup_total_bytes Total size for backup in progress.
# TYPE pgbackrest_stanza_backup_total_bytes gauge
pgbackrest_stanza_backup_total_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_complete_bytes Completed size for restore in progress.
# TYPE pgbackrest_stanza_restore_complete_bytes gauge
pgbackrest_stanza_restore_complete_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_lock_status Current stanza restore lock status.
# TYPE pgbackrest_stanza_restore_lock_status gauge
pgbackrest_stanza_restore_lock_status{stanza="demo"} 0
# HELP pgbackrest_stanza_restore_total_bytes Total size for restore in progress.
# TYPE pgbackrest_stanza_restore_total_bytes gauge
pgbackrest_stanza_restore_total_bytes{stanza="demo"} 0
# HELP pgbackrest_stanza_status Current stanza status.
# TYPE pgbackrest_stanza_status gauge
pgbackrest_stanza_status{stanza="demo"} 0
`
	tests := []struct {
		name string
		args args
	}{
		{
			"getStanzaMetricsBackupNotInProgress",
			args{
				templateStanzaRepoAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					2969514).Name,
				templateStanzaRepoAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					2969514).Status,
				templateStanzaRepoAbsent(
					"000000010000000000000004",
					"000000010000000000000001",
					2969514).Repo,
				setUpMetricValue,
				templateMetrics,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetMetrics()
			getStanzaMetrics(tt.args.stanzaName, tt.args.stanzaStatus, tt.args.stanzaRepo, tt.args.setUpMetricValueFun, logger)
			reg := prometheus.NewRegistry()
			reg.MustRegister(
				pgbrStanzaStatusMetric,
				pgbrStanzaBackupLockStatusMetric,
				pgbrStanzaBackupInProgressTotalMetric,
				pgbrStanzaBackupInProgressCompleteMetric,
				pgbrStanzaBackupInProgressRepoTotalMetric,
				pgbrStanzaBackupInProgressRepoCompleteMetric,
				pgbrStanzaRestoreLockStatusMetric,
				pgbrStanzaRestoreInProgressTotalMetric,
				pgbrStanzaRestoreInProgressCompleteMetric,
			)
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
		stanzaStatus        status
		stanzaRepo          *[]repo
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
					true,
					true,
					12,
					100,
					12345,
					1234,
					12345,
					1234,
					annotation{"testkey": "testvalue"}).Name,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					true,
					true,
					12,
					100,
					12345,
					1234,
					12345,
					1234,
					annotation{"testkey": "testvalue"}).Status,
				templateStanza(
					"000000010000000000000004",
					"000000010000000000000001",
					[]databaseRef{{"postgres", 13425}},
					true,
					true,
					true,
					12,
					100,
					12345,
					1234,
					12345,
					1234,
					annotation{"testkey": "testvalue"}).Repo,
				fakeSetUpMetricValue,
				9,
				9,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			lc := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug}))
			getStanzaMetrics(tt.args.stanzaName, tt.args.stanzaStatus, tt.args.stanzaRepo, tt.args.setUpMetricValueFun, lc)
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
