---
provider: brave
mode: web
mode_adjusted: true
truncated: false
sources_count: 5
cached: false
---

**prometheus.scrape | Grafana Agent documentation**
**It is set to 0 in case anything goes wrong with the scrape target**, either because it is not reachable, because the connection times out while scraping, or because the samples from the target could not be processed.

**Handle missing data in Grafana Alerting | Grafana documentation**
**Use absent_over_time() or present_over_time in Prometheus to detect when a metric or target disappears**. If data is frequently missing due to scrape delays, use techniques to account for data delays:

**Troubleshooting Managed Service for Prometheus | Google Cloud Observability**
**Check if you can query the metrics in the Google Cloud console but not Grafana**. If so, then the problem is with Grafana, not the collection of your metrics. Verify that the managed collector is able to scrape the exporter by inspecting the ...

**monitoring - How do I troubleshoot missing data in my Prometheus database? - DevOps Stack Exchange**
If you are using Java this can happen. Our fix was to **explicitly use the G1 garbage collector (Java 8+) with a specified limit on the length of GC activity to prevent these time gaps in the data collection**.

**prometheus.scrape | Grafana Alloy documentation**
It’s set to 0 in case anything goes wrong with the scrape target, either because it’s not reachable, because the connection times out while scraping, or because the samples from the target couldn’t be processed. When the target is behaving normally, the up metric is set to 1. To enable scraping of Prometheus’ native histograms over gRPC, the scrape_protocols should specify PrometheusProto as the first protocol to negotiate, for example: