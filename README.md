# prometheus-nagios-exporter

![CircleCI](https://img.shields.io/circleci/project/github/Financial-Times/prometheus-nagios-exporter/master.svg)

🆙 Prometheus exporter that scrapes a Nagios status pages for alerts.

Prometheus asks this exporter for metrics, one Nagios target at a time.

### Prometheus Configuration

```yaml
- job_name: nagios_exporter
  scheme: https
  static_configs:
    - targets:
      - prometheus-nagios-exporter-eu-west-1.in.ft.com
      - prometheus-nagios-exporter-us-east-1.in.ft.com
      labels:
        system: prometheus-nagios-exporter
        observe: yes

- job_name: nagios
  scheme: https
  metrics_path: /collect
  static_configs:
    - targets:
      - 10.0.0.1
      - 10.0.0.2
      - 10.0.0.3
      labels:
        observe: yes
        system: an-example-system-code
  relabel_configs:
    - source_labels: [__address__]
      target_label: __param_instance
    - source_labels: [__address__]
      target_label: instance
    - target_label: __address__
      replacement: prometheus-nagios-exporter.in.ft.com
```

## Development

### CircleCI

Ensure the following variables are set in the CircleCI project:

* `DOCKER_REGISTRY_USERNAME`
* `DOCKER_REGISTRY_PASSWORD`

### Local Development

Use the Makefile to locally test your changes. The 'make build' command will create a new local docker image, then the 'make run' command will execute these changes locally.

Go to `http://localhost:9942/` (or the port specifed on the `make run` command) on your browser to see the index page for the exporter.
