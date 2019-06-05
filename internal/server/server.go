package server

import (
	"log"
	"net/http"

	"github.com/Financial-Times/prometheus-nagios-exporter/internal/handlers"
	"github.com/Financial-Times/prometheus-nagios-exporter/internal/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

func Server(listenAddress string, httpClient *http.Client) *http.Server {
	router := http.NewServeMux()

	router.Handle("/", handlers.Index())
	router.Handle("/__gtg", handlers.GoodToGo())
	router.Handle("/metrics", promhttp.Handler())
	router.Handle("/collect", handlers.Collect(httpClient))

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
