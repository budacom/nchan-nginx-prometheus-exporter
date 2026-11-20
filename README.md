# Nchan Prometheus Exporter

Exports [Nchan](https://nchan.io) `nchan_stub_status` for Prometheus.

This image replaces `us-east1-docker.pkg.dev/ops-support-191021/platform/nchan-prometheus-exporter:0.2.0`. It scrapes Nchan only. Nginx `stub_status` is scraped by the official [nginx/nginx-prometheus-exporter](https://github.com/nginx/nginx-prometheus-exporter) as a second sidecar. Both containers share the pod network, so they cannot both listen on `:9113`.

## Run

```console
nchan-prometheus-exporter -scrape-uri http://127.0.0.1:81/nchan_status
```

The process listens on `:9113` and serves `/metrics`.

`-nginx` and `-nginx.scrape-uri` are still accepted, and ignored, so the current k8s args start this process. Nginx series from this image stop. Add the official exporter when those series are still needed:

```yaml
- name: nginx-prometheus-exporter
  image: nginx/nginx-prometheus-exporter:1.5.1
  args:
    - --nginx.scrape-uri=http://127.0.0.1:81/nginx_status
    - --web.listen-address=:9114
  ports:
    - name: nginx-metrics
      containerPort: 9114
```

The nchan ServiceMonitor scrapes the port named `metrics` (`9113`) only. Nginx metrics stay uncollected until that service and ServiceMonitor gain a second port.

## Flags

The process uses the current Prometheus exporter flags (`kingpin`, `exporter-toolkit`, `promslog`). Single-dash flags from the 0.2.0 image still work. Run `-h` for the full list.

```text
-scrape-uri string
    URI of the nchan_stub_status page (default "http://127.0.0.1:81/nchan_status")
-web.listen-address string
    Address to listen on (default ":9113", env LISTEN_ADDRESS)
-web.telemetry-path string
    Metrics path (default "/metrics", env TELEMETRY_PATH)
-web.config.file string
    exporter-toolkit web config (TLS and basic auth)
-ssl-verify
    Verify the server certificate when scrape-uri is https (default true)
-nginx
    Ignored
-nginx.scrape-uri string
    Ignored
```

## Metrics

Names and labels match the 0.2.0 exporter. There are no labels, except `version` on the info metric. Shared memory values are kilobytes: the trailing `K` is removed and the number is not converted to bytes.

| Name | Type |
| --- | --- |
| `nchan_messages_published_total` | counter |
| `nchan_messages_stored` | counter |
| `nchan_shared_memory_used` | gauge |
| `nchan_shared_memory_limit` | gauge |
| `nchan_channels` | gauge |
| `nchan_subscribers` | gauge |
| `nchan_redis_pending_commands` | gauge |
| `nchan_redis_connected_servers` | gauge |
| `nchan_interprocess_alerts_received_total` | counter |
| `nchan_interprocess_alerts_in_transit` | gauge |
| `nchan_interprocess_queued_alerts` | gauge |
| `nchan_interprocess_send_delay_total` | counter |
| `nchan_interprocess_receive_delay_total` | counter |

Nchan 1.3.9 adds three lines. Unknown lines are ignored.

| Name | Type |
| --- | --- |
| `nchan_redis_unhealthy_upstreams` | gauge |
| `nchan_redis_commands_sent_total` | counter |
| `nchan_version_info` | gauge, label `version` |
| `nchan_up` | gauge, 1 after a successful scrape |

A failed scrape logs the error, sets `nchan_up` to 0, and sends no other samples. The process stays up. `nchan_exporter_build_info` reports the exporter build.

## Image

Publishing a GitHub release pushes `us-east1-docker.pkg.dev/ops-support-191021/platform/nchan-prometheus-exporter:<release tag>`. The workflow uses the existing org variables `GOOGLE_WORKLOAD_IDENTITY_GITHUB_PROVIDER` and `GOOGLE_IMAGE_PUSHER_SA`. A bare tag push does not build the image.

The image-pusher service account has to be allowed to impersonate from `budacom/nchan-prometheus-exporter`. This repo does not create that binding.
