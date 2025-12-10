package main

import (
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	version_collector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web/kingpinflag"
	"github.com/woblerr/pgbackrest_exporter/backrest"
)

const exporterName = "pgbackrest_exporter"

func main() {
	var (
		webPath = kingpin.Flag(
			"web.telemetry-path",
			"Path under which to expose metrics.",
		).Default("/metrics").String()
		webAdditionalToolkitFlags = kingpinflag.AddFlags(kingpin.CommandLine, ":9854")
		collectionInterval        = kingpin.Flag(
			"collect.interval",
			"Collecting metrics interval in seconds.",
		).Default("600").Int()
		backrestCustomConfig = kingpin.Flag(
			"backrest.config",
			"Full path to pgBackRest configuration file.",
		).Default("").String()
		backrestCustomConfigIncludePath = kingpin.Flag(
			"backrest.config-include-path",
			"Full path to additional pgBackRest configuration files.",
		).Default("").String()
		backrestIncludeStanza = kingpin.Flag(
			"backrest.stanza-include",
			"Specific stanza for collecting metrics. Can be specified several times.",
		).Default("").PlaceHolder("\"\"").Strings()
		backrestExcludeStanza = kingpin.Flag(
			"backrest.stanza-exclude",
			"Specific stanza to exclude from collecting metrics. Can be specified several times.",
		).Default("").PlaceHolder("\"\"").Strings()
		backrestBackupType = kingpin.Flag(
			"backrest.backup-type",
			"Specific backup type for collecting metrics. One of: [full, incr, diff].",
		).Default("").String()
		backrestBackupDBCount = kingpin.Flag(
			"backrest.database-count",
			"Exposing the number of databases in backups.",
		).Default("false").Bool()
		backrestBackupDBCountParallelProcesses = kingpin.Flag(
			"backrest.database-parallel-processes",
			"Number of parallel processes for collecting information about databases.",
		).Default("1").Int()
		backrestBackupDBCountLatest = kingpin.Flag(
			"backrest.database-count-latest",
			"Exposing the number of databases in the latest backups.",
		).Default("false").Bool()
		backrestBackupReferenceCount = kingpin.Flag(
			"backrest.reference-count",
			"Exposing the number of references to other backups (backup reference list).",
		).Default("false").Bool()
		backrestVerboseWAL = kingpin.Flag(
			"backrest.verbose-wal",
			"Exposing additional labels for WAL metrics.",
		).Default("false").Bool()
	)
	// Set logger config.
	promslogConfig := &promslog.Config{}
	// Add flags log.level and log.format from promlog package.
	flag.AddFlags(kingpin.CommandLine, promslogConfig)
	kingpin.Version(version.Print(exporterName))
	// Add short help flag.
	kingpin.HelpFlag.Short('h')
	// Load command line arguments.
	kingpin.Parse()
	// Setup signal catching.
	sigs := make(chan os.Signal, 1)
	// Catch  listed signals.
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	// Set logger.
	logger := promslog.New(promslogConfig)
	// Method invoked upon seeing signal.
	go func(logger *slog.Logger) {
		s := <-sigs
		logger.Warn(
			"Stopping exporter",
			"name", filepath.Base(os.Args[0]),
			"signal", s)
		os.Exit(1)
	}(logger)
	logger.Info(
		"Starting exporter",
		"name", filepath.Base(os.Args[0]),
		"version", version.Info())
	logger.Info("Build context", "build_context", version.BuildContext())
	resetMetricsAfterFetch := strings.Join(*backrestIncludeStanza, "") == ""
	// Create BackrestExporterConfig from flags.
	backrestExporterConfig := backrest.BackrestExporterConfig{
		Config:                         *backrestCustomConfig,
		ConfigIncludePath:              *backrestCustomConfigIncludePath,
		BackupType:                     *backrestBackupType,
		IncludeStanza:                  *backrestIncludeStanza,
		ExcludeStanza:                  *backrestExcludeStanza,
		BackupReferenceCount:           *backrestBackupReferenceCount,
		BackupDBCount:                  *backrestBackupDBCount,
		BackupDBCountLatest:            *backrestBackupDBCountLatest,
		VerboseWAL:                     *backrestVerboseWAL,
		ResetMetricsAfter:              resetMetricsAfterFetch,
		BackupDBCountParallelProcesses: *backrestBackupDBCountParallelProcesses,
	}
	// Log BackrestExporterConfig parameters.
	backrest.LogBackrestExporterConfig(backrestExporterConfig, logger)
	// Setup parameters for exporter.
	backrest.SetPromPortAndPath(*webAdditionalToolkitFlags, *webPath)
	logger.Info(
		"Use exporter parameters",
		"endpoint", *webPath,
		"config.file", *webAdditionalToolkitFlags.WebConfigFile,
	)
	// Exporter build info metric
	prometheus.MustRegister(version_collector.NewCollector(exporterName))
	// Start web server.
	backrest.StartPromEndpoint(version.Info(), logger)
	for {
		// Get pgBackRest version info and set metric.
		backrest.GetPgBackrestVersionInfo(logger)
		// Get information form pgBackRest and set metrics.
		backrest.GetPgBackRestInfo(backrestExporterConfig, logger)
		// Sleep for 'collection.interval' seconds.
		time.Sleep(time.Duration(*collectionInterval) * time.Second)
	}
}
