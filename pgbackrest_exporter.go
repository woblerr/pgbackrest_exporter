package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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
		verboseInfo = kingpin.Flag(
			"verbose.info",
			"Enable additional metrics labels.",
		).Default("false").Bool()
	)
	// Load command line arguments.
	kingpin.Parse()
	// Setup signal catching.
	sigs := make(chan os.Signal, 1)
	// Catch  listed signals.
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	// Method invoked upon seeing signal.
	go func() {
		s := <-sigs
		log.Printf("[WARN] RECEIVED SIGNAL %s", s)
		log.Printf("[WARN] Stopping  %s", filepath.Base(os.Args[0]))
		os.Exit(1)
	}()
	log.Printf("[INFO] Starting %s", filepath.Base(os.Args[0]))
	log.Printf("[INFO] Version %s", version)
	log.Printf("[INFO] Verbose info %t", *verboseInfo)
	log.Printf("[INFO] Collecting metrics every %d seconds", *collectionInterval)
	if *backrestCustomConfig != "" {
		log.Printf("[INFO] Custom pgBackRest configuration file %s", *backrestCustomConfig)
	}
	if *backrestCustomConfigIncludePath != "" {
		log.Printf("[INFO] Custom path to additional pgBackRest configuration files %s", *backrestCustomConfigIncludePath)
	}
	if strings.Join(*backrestIncludeStanza, "") != "" {
		for _, stanza := range *backrestIncludeStanza {
			log.Printf("[INFO] Collecting metrics for specific stanza %s", stanza)
		}
	}
	if strings.Join(*backrestExcludeStanza, "") != "" {
		for _, stanza := range *backrestExcludeStanza {
			log.Printf("[INFO] Exclude collecting metrics for specific stanza %s", stanza)
		}
	}
	// Setup parameters for exporter.
	backrest.SetPromPortandPath(*promPort, *promPath)
	log.Printf("[INFO] Use port %s and HTTP endpoint %s", *promPort, *promPath)
	// Start exporter.
	backrest.StartPromEndpoint()
	for {
		// Reset metrics.
		backrest.ResetMetrics()
		// Get information form pgBackRest.
		backrest.GetPgBackRestInfo(
			*backrestCustomConfig,
			*backrestCustomConfigIncludePath,
			*backrestIncludeStanza,
			*backrestExcludeStanza,
			*verboseInfo,
		)
		// Sleep for 'collection.interval' seconds.
		time.Sleep(time.Duration(*collectionInterval) * time.Second)
	}
}
