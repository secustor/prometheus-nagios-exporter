package server

import (
	"log"
	"net/http"

	"github.com/itsdone/prometheus-nagios-exporter/pkg/handlers"
	"github.com/itsdone/prometheus-nagios-exporter/pkg/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

func Server(listenAddress string, httpClient *http.Client, basicAuthUsername string, basicAuthPassword string) *http.Server {
	router := http.NewServeMux()

	router.Handle("/", handlers.Index())
	router.Handle("/healthz", handlers.Healthz())
	router.Handle("/metrics", promhttp.Handler())
	router.Handle("/collect", handlers.Collect(httpClient, basicAuthUsername, basicAuthPassword))

	logger := logrus.New()
	w := logger.Writer()
	defer w.Close()

	server := &http.Server{
		Addr:     listenAddress,
		Handler:  middleware.Logging()(router),
		ErrorLog: log.New(w, "", 0),
	}

	return server
}
