package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/Financial-Times/prometheus-nagios-exporter/internal/collectors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

func Collect() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		instance := r.URL.Query().Get("instance")

		if instance == "" {
			log.WithFields(log.Fields{
				"event": "ERROR_COLLECT_MISSING_INSTANCE",
			}).Error("Request is missing an instance parameter.")

			http.Error(w, "Request is missing an instance parameter.", http.StatusBadRequest)

			return
		}

		// Set a default timeout of 10 seconds.
		var timeout float64 = 10

		// Offset to subtract from timeout in seconds, ensures this exporter will respond to Prometheus requests.
		timeout -= 0.5

		// Add the timeout to this request.
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout*float64(time.Second)))
		defer cancel()
		r = r.WithContext(ctx)

		collector := collectors.NewNagiosCollector(instance)
		registry := prometheus.NewPedanticRegistry()
		registry.MustRegister(collector)

		handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		handler.ServeHTTP(w, r)
	})
}
