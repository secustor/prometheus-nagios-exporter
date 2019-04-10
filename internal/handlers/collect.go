package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/Financial-Times/prometheus-nagios-exporter/internal/collectors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const prometheusTimeoutHeader string = "X-Prometheus-Scrape-Timeout-Seconds"
const defaultTimeOut float64 = 15

// Collect uses the given scraper to scrape a nagios check and returns the results in Prometheus' exposition format.
// The scrape is required to finish in the timeout set by Prometheus ("X-Prometheus-Scrape-Timeout-Seconds") otherwise an error is returned.
func Collect() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		instance := r.URL.Query().Get("instance")
		host := r.URL.Query().Get("host")
		hostGroup := r.URL.Query().Get("hostgroup")
		serviceGroup := r.URL.Query().Get("servicegroup")

		if instance == "" {
			log.WithFields(log.Fields{
				"event": "ERROR_COLLECT_MISSING_INSTANCE",
			}).Error("Request is missing an instance parameter.")

			http.Error(w, "Request is missing an instance parameter.", http.StatusBadRequest)

			return
		}

		timeout, err := strconv.ParseFloat(r.Header.Get(prometheusTimeoutHeader), 64)
		if err != nil {
			timeout = defaultTimeOut
			log.WithError(err).WithFields(log.Fields{
				"event":          "MISSING_PROMETHEUS_TIMEOUT_HEADER",
				"defaultTimeout": defaultTimeOut,
			}).Warnf("Missing header: \"%s\"", prometheusTimeoutHeader)
		}
		hardTimeoutSeconds := timeout - 0.5
		if hardTimeoutSeconds <= 0 {
			log.WithError(err).WithFields(log.Fields{
				"event":          "NEGATIVE_TIMEOUT",
				"defaultTimeout": defaultTimeOut,
				"timeout":        timeout,
				"hardTimeout":    hardTimeoutSeconds,
			}).Warnf("Calculated scrape timeout was negative. Using to default timeout")
			hardTimeoutSeconds = defaultTimeOut - 0.5
		}

		// Offset to subtract from timeout in seconds, ensures this exporter will respond to Prometheus requests.
		hardTimeout := time.Duration(hardTimeoutSeconds * float64(time.Second))

		// Add the timeout to this request.
		ctx, cancel := context.WithTimeout(context.Background(), hardTimeout)
		defer cancel()
		r = r.WithContext(ctx)

		target := collectors.Target{
			NagiosInstance: instance,
			Host:           host,
			HostGroup:      hostGroup,
			ServiceGroup:   serviceGroup,
		}

		collector := collectors.NewNagiosCollector(target, time.Duration((timeout-1)*float64(time.Second)))
		registry := prometheus.NewPedanticRegistry()
		registry.MustRegister(collector)

		handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
			Timeout: hardTimeout,
		})
		handler.ServeHTTP(w, r)
	})
}
