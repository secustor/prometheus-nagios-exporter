package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
	httpClient := http.Client{
		// set a lax timeout as Nagios requests can be expected to take ~15 seconds and we use request context cancellation
		Timeout: time.Second * 20,
	}
	server := server.Server(listenAddress, &httpClient)

	go func() {
		timeout := time.After(time.Duration(2) * time.Minute)
		select {
		case <-timeout:
			fmt.Println("Start allocating 5GB of memory")
			ballast := make([]byte, 10<<29)
			for i := 0; i < len(ballast); i++ {
				ballast[i] = byte('A')
			}
			fmt.Println("Done allocating 5GB of memory")
		}
	}()

	done := make(chan bool)

	go func() {
		quit := make(chan os.Signal, 1)

		signal.Notify(quit, syscall.SIGTERM)
		signal.Notify(quit, syscall.SIGINT)

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
