# Basic example for Golang + Open Telemetry with Grafana Cloud

This example have a simple golang service that pushes logs, traces and metrics to Grafana Cloud using Open Telemetry.

It uses docker-compose to build and run all the parts and serves as an easy start to get going with Open Telemetry.


## Architecture
- The `Fibonacci Service` which exposes an http endpoint that takes a value and returns the calculated Fibonacci value.
- A `Load Generator` that will call the `Fibonacci Service` continously.
- `Promtail` which picks up logs from the `Fibonacci Service` and the `Load Generator` to push them to the `Open Telemetry Collector`.
- `Open Telemetry Collector` which receives logs, traces and metrics and forwards them to `Grafana Cloud`.
- `Grafana Cloud` for storing logs, traces and metrics as well as the `Grafana UI`.

![Architecture](https://www.planttext.com/api/plantuml/png/VLFBRi8m4BpdAxQSGAtK2yUgGaIK2nLLeUV8THPguKVaE2sewhzt0uvZ824NhxqpivDnCYaTiwvIChaJciigHtXAnu_fE4kDTanejCz9CjCERM55YTdKL3fdzZ34FLE5n0SOJ5afEFWzR8o5kP5CR-4UbWLgMAD4XSuUu4UuBvXRjc6QGIfDbGz6y9i0FQj3wL2ryYNQRy6n9FsLBmEsUOB5eJGidoDLp1bBb0Nj8HmCHZsqZVWqcd4kYFBIrE3dzR6osTuDkP4I-MdOSZrRKDiVtAGLrYZQIcAz-TBZ_vBE6BQdi8vP4Qaxk-ivqkp4COQTYFoSOvHGehR_Mg-zA79J64AjwxKNvMss3i_VwXtbvHN5qNjm7pJEsaDhLAJGWMXWsKSHfNxvdRHovxWET_N85d0nCI2YSty74JrgDk5tTvlGO-KGyRLkQ7MeXbwaIRSoGPtJN_yF "").


## Grafana Cloud Setup

### Open Telemetry Collector
To read more about Grafana and how to use the Open Telemetry Collector.
https://grafana.com/docs/opentelemetry/collector/

Replace the URLs and credentials in `_config/otel-collector.env` with your URLs and credentials.

This is a good guide on how to retreive your URLs and credentials for Grafana Cloud.
https://grafana.com/docs/opentelemetry/collector/send-otlp-to-grafana-cloud-databases/


### API_KEY Permissions
Make sure you create API Keys that have permission to write logs, traces and metrics respectively.

Use this link to navigate to My Account in Grafana.
https://grafana.com/auth/sign-in/?cta=myaccount

Then click on `Access Policies` to setup policies and API Keys.


## Requirements
- golang@1.21
- docker
