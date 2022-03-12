package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/woblerr/pgbackrest_exporter/backrest"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var version = "unknown"

func main() {
	var (
		promPort = kingpin.Flag(
			"prom.port",
			"Port for prometheus metrics to listen on.",
		).Default("9854").String()
		promPath = kingpin.Flag(
			"prom.endpoint",
			"Endpoint used for metrics.",
		).Default("/metrics").String()
		promTLSConfigFile = kingpin.Flag(
			"prom.web-config",
			"[EXPERIMENTAL] Path to config yaml file that can enable TLS or authentication.",
		).Default("").String()
		collectionInterval = kingpin.Flag(
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
			"Specific backup type for collecting metrics.",
		).Default("").String()
		verboseInfo = kingpin.Flag(
			"verbose.info",
			"Enable additional metrics labels.",
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
		"version", version,
		"verbose.info", *verboseInfo)
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
	// Setup parameters for exporter.
	backrest.SetPromPortandPath(*promPort, *promPath, *promTLSConfigFile)
	level.Info(logger).Log(
		"mgs", "Use port and HTTP endpoint",
		"port", *promPort,
		"endpoint", *promPath,
		"web-config", *promTLSConfigFile,
	)
	// Start exporter.
	backrest.StartPromEndpoint(logger)
	// Set up exporter info metric.
	// There is no need to reset metric every time,
	// it is set up once at startup.
	backrest.GetExporterInfo(version, logger)
	for {
		// Reset metrics.
		backrest.ResetMetrics()
		// Get information form pgBackRest.
		backrest.GetPgBackRestInfo(
			*backrestCustomConfig,
			*backrestCustomConfigIncludePath,
			*backrestIncludeStanza,
			*backrestExcludeStanza,
			*backrestBackupType,
			*verboseInfo,
			logger,
		)
		// Sleep for 'collection.interval' seconds.
		time.Sleep(time.Duration(*collectionInterval) * time.Second)
	}
}
