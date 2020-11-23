package collectors

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type nagiosCollector struct {
	ctx         context.Context
	netClient   *http.Client
	target      Target
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

type Target struct {
	NagiosInstance string
	Host           string
	HostGroup      string
	ServiceGroup   string
	Protocol       string
	BasicAuth      BasicAuth
}

type BasicAuth struct {
	Username string
	Password string
}

func NewNagiosCollector(ctx context.Context, netClient *http.Client, target Target) *nagiosCollector {
	if target.Host == "" {
		target.Host = "all"
	}
	if target.Protocol == "" {
		target.Protocol = "http"
	}

	return &nagiosCollector{
		ctx:       ctx,
		netClient: netClient,
		target:    target,
		checkStatus: prometheus.NewDesc(
			"nagios_check_ok",
			"Status of a service on a host monitored by a Nagios instance, 1 is OK.",
			[]string{"host", "check_id", "state", "notify", "acknowledged"},
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
			"Whether the last Nagios scrape was successful (1:up, 0:down).",
			nil,
			nil,
		),
	}
}

func (collector *nagiosCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.duration
	ch <- collector.up
	ch <- collector.checkStatus
}

// parseNagiosOutput parses the Nagios HTML response to a slice of Labels
func parseNagiosOutput(bodyReader io.Reader) ([]Label, error) {
	var instance string
	var checks []Label

	document, err := goquery.NewDocumentFromReader(bodyReader)

	if err != nil {
		return nil, err
	}

	table := document.Find("table.status > tbody > tr")

	// body > p > table.status > tbody > tr:nth-child(8) > td.statusEven > table > tbody > tr > td:nth-child(1) > table > tbody > tr > td > a
	for i := range table.Nodes {
		var serviceState string

		if i == 0 {
			continue
		}

		node := table.Eq(i)

		notifications := "true"
		acknowledged := "false"

		if host := node.Find("td:nth-of-type(1) > table > tbody > tr > td:nth-child(1) > table > tbody > tr > td > a").Text(); host != "" {
			instance = host
		}

		serviceName := node.Find("td:nth-of-type(2) > table > tbody > tr > td:nth-child(1) > table > tbody > tr > td > a").Text()

		node.Find("img").Each(func(index int, element *goquery.Selection) {
			imageSource, exists := element.Attr("src")
			if exists {
				if strings.Contains(imageSource, "/images/ndisabled.gif") {
					notifications = "false"
				}
				if strings.Contains(imageSource, "/images/ack.gif") {
					acknowledged = "true"
				}
			}
		})
		// Nagios outputs some empty rows for formatting ¯\_(ツ)_/¯
		if serviceName == "" {
			continue
		}

		switch node.Find("td:nth-of-type(3)").Text() {
		case "OK":
			serviceState = "ok"
		case "WARNING":
			serviceState = "warning"
		case "CRITICAL":
			serviceState = "critical"
		default:
			serviceState = "unknown"
		}

		checks = append(checks, Label{
			Host:          instance,
			CheckId:       serviceName,
			State:         serviceState,
			Notifications: notifications,
			Acknowledged:  acknowledged,
		})
	}
	return checks, nil
}

// scrape scrapes the given target with the given netclient and translates the Nagios output to a set of Prometheus labels.
func (collector *nagiosCollector) scrape(netClient *http.Client, target Target) ([]Label, error) {
	nagiosUrl, err := url.Parse(fmt.Sprintf("%s://%s/nagios/cgi-bin/status.cgi?embedded=1&noheader=1&limit=all&style=detail", target.Protocol, target.NagiosInstance))
	if err != nil {
		return nil, err
	}

	query := nagiosUrl.Query()
	if target.ServiceGroup != "" {
		query.Set("servicegroup", target.ServiceGroup)
	} else {
		if target.HostGroup != "" {
			query.Set("hostgroup", target.HostGroup)
		} else {
			query.Set("host", target.Host)
		}
	}

	nagiosUrl.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, nagiosUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "prometheus-nagios-exporter")

	if target.BasicAuth.Username != "" && target.BasicAuth.Password != "" {
		req.SetBasicAuth(target.BasicAuth.Username, target.BasicAuth.Password)
	}

	req = req.WithContext(collector.ctx)
	res, err := netClient.Do(req)

	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		var errorCodeResponse = fmt.Errorf("instance returns HTTP code %s", res.Status)
		return nil, errorCodeResponse
	}
	defer res.Body.Close()

	return parseNagiosOutput(res.Body)
}

func (collector *nagiosCollector) Collect(ch chan<- prometheus.Metric) {
	// Fetch all checks per instance/host.
	start := time.Now()

	checks, err := collector.scrape(collector.netClient, collector.target)

	if err != nil {
		log.WithFields(log.Fields{
			"event":    "ERROR_NAGIOS_SCRAPE",
			"instance": collector.target.NagiosInstance,
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
