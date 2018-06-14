package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"

	"github.com/Financial-Times/prometheus-nagios-exporter/internal/server"
	"golang.org/x/crypto/ssh/terminal"

	log "github.com/sirupsen/logrus"
)

var (
	listenAddress string
	healthy       int32
	verbose       bool
)

func main() {
	flag.StringVar(&listenAddress, "web.listen-address", ":9942", "Address to listen on for web interface and telemetry.")
	flag.BoolVar(&verbose, "verbose", false, "Enable more detailed logging.")
	flag.Parse()

	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		log.SetFormatter(&log.JSONFormatter{})
	}

	server := server.Server(listenAddress)

	done := make(chan bool)

	go func() {
		quit := make(chan os.Signal, 1)

		signal.Notify(quit, os.Interrupt)

		<-quit

		if err := server.Close(); err != nil {
			log.WithFields(log.Fields{
				"event": "ERROR_STOPPING",
				"err":   err,
			}).Fatal("Could not gracefully stop the nagios exporter.")
		}

		close(done)
	}()

	log.WithFields(log.Fields{
		"event":         "STARTED",
		"listenAddress": listenAddress,
	}).Info("nagios exporter is ready to handle requests.")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.WithFields(log.Fields{
			"event":         "ERROR_STARTING",
			"listenAddress": listenAddress,
			"err":           err,
		}).Fatal("Could not listen at the specified address.")
	}

	<-done

	log.WithField("event", "STOPPED").Info("nagios exporter has stopped.")
}
