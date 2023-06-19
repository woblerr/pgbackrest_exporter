package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/exporter-toolkit/web/kingpinflag"
	"github.com/woblerr/pgbackrest_exporter/backrest"
)

var version = "unknown"

func main() {
	var (
		webPath = kingpin.Flag(
			"web.endpoint",
			"Endpoint used for metrics.",
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
		backrestVerboseWAL = kingpin.Flag(
			"backrest.verbose-wal",
			"Exposing additional labels for WAL metrics.",
		).Default("false").Bool()
	)
	// Set logger config.
	promlogConfig := &promlog.Config{}
	// Add flags log.level and log.format from promlog package.
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	// Add short help flag.
	kingpin.HelpFlag.Short('h')
	// Load command line arguments.
	kingpin.Parse()
	// Setup signal catching.
	sigs := make(chan os.Signal, 1)
	// Catch  listed signals.
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	// Set logger.
	logger := promlog.New(promlogConfig)
	// Method invoked upon seeing signal.
	go func(logger log.Logger) {
		s := <-sigs
		level.Warn(logger).Log(
			"msg", "Stopping exporter",
			"name", filepath.Base(os.Args[0]),
			"signal", s)
		os.Exit(1)
	}(logger)
	level.Info(logger).Log(
		"msg", "Starting exporter",
		"name", filepath.Base(os.Args[0]),
		"version", version)
	if *backrestCustomConfig != "" {
		level.Info(logger).Log(
			"mgs", "Custom pgBackRest configuration file",
			"file", *backrestCustomConfig)
	}
	if *backrestCustomConfigIncludePath != "" {
		level.Info(logger).Log(
			"mgs", "Custom path to additional pgBackRest configuration files",
			"path", *backrestCustomConfigIncludePath)
	}
	if strings.Join(*backrestIncludeStanza, "") != "" {
		backrest.MetricResetFlag = false
		for _, stanza := range *backrestIncludeStanza {
			level.Info(logger).Log(
				"mgs", "Collecting metrics for specific stanza",
				"stanza", stanza)
		}
	}
	if strings.Join(*backrestExcludeStanza, "") != "" {
		for _, stanza := range *backrestExcludeStanza {
			level.Info(logger).Log(
				"mgs", "Exclude collecting metrics for specific stanza",
				"stanza", stanza)
		}
	}
	if *backrestBackupType != "" {
		level.Info(logger).Log(
			"mgs", "Collecting metrics for specific backup type",
			"type", *backrestBackupType)
	}
	if *backrestBackupDBCount {
		level.Info(logger).Log(
			"msg", "Exposing the number of databases in backups",
			"database-count", *backrestBackupDBCount,
			"database-parallel-processes", *backrestBackupDBCountParallelProcesses)
	}
	if *backrestBackupDBCountLatest {
		level.Info(logger).Log(
			"msg", "Exposing the number of databases in the latest backups",
			"database-count-latest", *backrestBackupDBCountLatest)
	}
	if *backrestVerboseWAL {
		level.Info(logger).Log(
			"mgs", "Enabling additional labels for WAL metrics",
			"verbose-wal", *backrestVerboseWAL)
	}
	// Setup parameters for exporter.
	backrest.SetPromPortAndPath(*webAdditionalToolkitFlags, *webPath)
	level.Info(logger).Log(
		"mgs", "Use exporter parameters",
		"endpoint", *webPath,
		"config.file", *webAdditionalToolkitFlags.WebConfigFile,
	)
	// Start exporter.
	backrest.StartPromEndpoint(logger)
	// Set up exporter info metric.
	// There is no need to reset metric every time,
	// it is set up once at startup.
	backrest.GetExporterInfo(version, logger)
	for {
		// Get information form pgBackRest and set metrics.
		backrest.GetPgBackRestInfo(
			*backrestCustomConfig,
			*backrestCustomConfigIncludePath,
			*backrestBackupType,
			*backrestIncludeStanza,
			*backrestExcludeStanza,
			*backrestBackupDBCount,
			*backrestBackupDBCountLatest,
			*backrestVerboseWAL,
			*backrestBackupDBCountParallelProcesses,
			logger,
		)
		// Sleep for 'collection.interval' seconds.
		time.Sleep(time.Duration(*collectionInterval) * time.Second)
	}
}
