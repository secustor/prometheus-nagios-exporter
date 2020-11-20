package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/itsdone/prometheus-nagios-exporter/internal/server"
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
	pflag.StringP("username", "U", "", "(optional) username for basic auth")
	pflag.StringP("password", "P", "", "(optional) password for basic auth")
	pflag.BoolP("insecure", "k", false, "ignore TLS error when connection to Nagios instances")
	pflag.IntP("client-timeout", "t", 60, "hard timeout for http requests")
	pflag.Parse()

	viper.BindPFlags(pflag.CommandLine)

	if viper.GetBool("verbose") {
		log.SetLevel(log.DebugLevel)
	}

	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		log.SetFormatter(&log.JSONFormatter{})
	}

	var username, password string
	if viper.GetString("username") != "" && viper.GetString("password") != "" {
		username = viper.GetString("username")
		password = viper.GetString("password")
	}
	listenAddress := fmt.Sprintf(":%d", viper.GetInt("port"))
	httpClient := http.Client{
		Timeout: time.Second * time.Duration(viper.GetInt("client-timeout")),
	}

	if viper.GetBool("insecure") {
		transCfg := http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // ignore expired SSL certificates
			},
		}
		httpClient.Transport = &transCfg
	}

	server := server.Server(listenAddress, &httpClient, username, password)

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
