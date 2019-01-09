package collectors

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type nagiosCollector struct {
	netClient  *http.Client
	target     string
	hostStatus *prometheus.Desc
	duration   *prometheus.Desc
	up         *prometheus.Desc
}

func NewNagiosCollector(target string, timeOut time.Duration) *nagiosCollector {
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	var netClient = &http.Client{
		Transport: netTransport,
		Timeout:   timeOut,
	}

	return &nagiosCollector{
		netClient: netClient,
		target:    target,
		hostStatus: prometheus.NewDesc(
			"nagios_host_ok",
			"Status of a host monitored by Nagios, 1 is OK.",
			[]string{"host"},
			nil,
		),
		duration: prometheus.NewDesc(
			"nagios_request_duration_seconds",
			"How long the exporter took to scrape the Nagios host.",
			nil,
			nil,
		),
		up: prometheus.NewDesc(
			"nagios_up",
			"Whether the last Nagios scrape was successful (1:up, 0:down) ",
			nil,
			nil,
		),
	}
}

func (collector *nagiosCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.hostStatus
	ch <- collector.duration
	ch <- collector.up
}

func Scrape(netClient *http.Client, target string) (map[string]float64, error) {
	res, err := netClient.Get(fmt.Sprintf("http://%s/nagios/cgi-bin/status.cgi?host=all&embedded=1&noheader=1&limit=all", target))

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	document, err := goquery.NewDocumentFromReader(res.Body)

	if err != nil {
		return nil, err
	}

	var instance string

	instances := make(map[string]float64)

	table := document.Find("table.status > tbody > tr")

	// body > p > table.status > tbody > tr:nth-child(8) > td.statusEven > table > tbody > tr > td:nth-child(1) > table > tbody > tr > td > a
	for i := range table.Nodes {
		if i == 0 {
			continue
		}

		node := table.Eq(i)

		if host := node.Find("td:nth-of-type(1) > table > tbody > tr > td:nth-child(1) > table > tbody > tr > td > a").Text(); host != "" {
			instance = host
		}

		name := node.Find("td:nth-of-type(2) > table > tbody > tr > td:nth-child(1) > table > tbody > tr > td > a").Text()

		// Nagios outputs some empty rows for formatting ¯\_(ツ)_/¯
		if name == "" {
			continue
		}
		var status float64
		switch node.Find("td:nth-of-type(3)").Text() {
		case "OK":
			status = 1
		default:
			status = 0
		}

		if val, exists := instances[instance]; !exists || val == 1 {
			instances[instance] = status
		}
	}

	return instances, nil
}

func (collector *nagiosCollector) Collect(ch chan<- prometheus.Metric) {
	// Fetch and record the health check results.
	start := time.Now()

	hosts, err := Scrape(collector.netClient, collector.target)

	if err != nil {
		log.WithFields(log.Fields{
			"event":    "ERROR_NAGIOS_SCRAPE",
			"instance": collector.target,
		}).Error(err)

		ch <- prometheus.MustNewConstMetric(
			collector.up,
			prometheus.GaugeValue,
			float64(0),
		)
		return
	}

	duration := time.Since(start).Seconds()

	ch <- prometheus.MustNewConstMetric(
		collector.duration,
		prometheus.GaugeValue,
		duration,
	)

	ch <- prometheus.MustNewConstMetric(
		collector.up,
		prometheus.GaugeValue,
		float64(1),
	)
	for host, status := range hosts {

		ch <- prometheus.MustNewConstMetric(
			collector.hostStatus,
			prometheus.GaugeValue,
			status,
			host,
		)
	}
}
