package collectors

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type nagiosCollector struct {
	netClient   *http.Client
	target      string
	hostStatus  *prometheus.Desc
	duration    *prometheus.Desc
	up          *prometheus.Desc
	checkStatus *prometheus.Desc
}
type Label struct {
	Host          string
	CheckId       string
	State         string
	Notifications string
	Acknowledged  string
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
		checkStatus: prometheus.NewDesc(
			"nagios_check_ok",
			"Status of a service on a host monitored by a Nagios instance, 1 is OK.",
			[]string{"host", "check_id", "state", "notifications", "acknowledged"},
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
	ch <- collector.checkStatus
}

func Scrape(netClient *http.Client, target string) (map[string]float64, []Label, error) {
	res, err := netClient.Get(fmt.Sprintf("http://%s/nagios/cgi-bin/status.cgi?host=all&embedded=1&noheader=1&limit=all", target))

	if err != nil {
		return nil, nil, err
	}

	defer res.Body.Close()

	document, err := goquery.NewDocumentFromReader(res.Body)

	if err != nil {
		return nil, nil, err
	}

	var instance string

	instances := make(map[string]float64)
	var checks []Label
	table := document.Find("table.status > tbody > tr")

	// body > p > table.status > tbody > tr:nth-child(8) > td.statusEven > table > tbody > tr > td:nth-child(1) > table > tbody > tr > td > a
	for i := range table.Nodes {
		if i == 0 {
			continue
		}

		node := table.Eq(i)

		notifications := "enabled"
		acknowledged := "no"

		if host := node.Find("td:nth-of-type(1) > table > tbody > tr > td:nth-child(1) > table > tbody > tr > td > a").Text(); host != "" {
			instance = host
		}

		serviceName := node.Find("td:nth-of-type(2) > table > tbody > tr > td:nth-child(1) > table > tbody > tr > td > a").Text()

		node.Find("img").Each(func(index int, element *goquery.Selection) {
			imageSource, exists := element.Attr("src")
			if exists {
				if strings.Contains(imageSource, "/images/ndisabled.gif") {
					notifications = "disabled"
				}
				if strings.Contains(imageSource, "/images/ack.gif") {
					acknowledged = "yes"
				}
			}
		})
		// Nagios outputs some empty rows for formatting ¯\_(ツ)_/¯
		if serviceName == "" {
			continue
		}
		var status float64
		var serviceState string
		switch node.Find("td:nth-of-type(3)").Text() {
		case "OK":
			status = 1
			serviceState = "ok"
		case "WARNING":
			serviceState = "warning"
		case "CRITICAL":
			serviceState = "critical"
		default:
			status = 0
			serviceState = "unknown"
		}
		if val, exists := instances[instance]; !exists || val == 1 {
			instances[instance] = status
		}
		checks = append(checks, Label{
			Host:          instance,
			CheckId:       serviceName,
			State:         serviceState,
			Notifications: notifications,
			Acknowledged:  acknowledged,
		})
	}
	return instances, checks, nil
}

func (collector *nagiosCollector) Collect(ch chan<- prometheus.Metric) {
	// Fetch all checks per instance/host.
	start := time.Now()

	hosts, checks, err := Scrape(collector.netClient, collector.target)

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
	for _, labels := range checks {
		checkStatus := 0
		if labels.State == "ok" {
			checkStatus = 1
		}
		ch <- prometheus.MustNewConstMetric(
			collector.checkStatus,
			prometheus.GaugeValue,
			float64(checkStatus),
			labels.Host,
			labels.CheckId,
			labels.State,
			labels.Notifications,
			labels.Acknowledged,
		)
	}
}
