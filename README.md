# Zhone-exporter

## Overview

Zhone Exporter is a basic Prometheus exporter for the Zhone ZNID-GPON-2726A1-UK gateway. The gateway does not provide an SNMP interface, and as such, the metrics are gathered through web scraping.

## Use
`zhone-exporter $ENDPOINT`

A sample systemd unit file is also provided in [zhone-exporter.service](zhone-exporter.service)
`zhone-exporter.service`

## Example Dashboards
2 sample dashboards are provided in the [Dashboards](Dashboards/) subdirectory:

### Dashboard 1: Interface overview
![Grafana Interfaces Dashboard](Dashboards/Dashboard_Interfaces.png?raw=true)

### Dashboard 2: Wifi metrics overview
![Grafana Wifi Dashboard](Dashboards/Dashboard_Wifi.png?raw=true)