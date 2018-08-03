package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/Financial-Times/prometheus-nagios-exporter/internal/server"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"

	log "github.com/sirupsen/logrus"
)

func main() {
	viper.AutomaticEnv()
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	pflag.IntP("port", "p", 8080, "Port to listen on")
	pflag.BoolP("verbose", "v", false, "Enable more detailed logging.")
	pflag.Parse()

	viper.BindPFlags(pflag.CommandLine)

	if viper.GetBool("verbose") {
		log.SetLevel(log.DebugLevel)
	}

	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		log.SetFormatter(&log.JSONFormatter{})
	}

	listenAddress := fmt.Sprintf(":%d", viper.GetInt("port"))

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
