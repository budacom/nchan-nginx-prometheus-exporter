# Changelog

## Unreleased

- Parse `nchan_stub_status` as `key: value` lines, including the Nchan 1.3.9 fields.
- Use the current Prometheus exporter stack: client_golang v1.25, exporter-toolkit v0.20, and promslog.
- Export Nchan only. Nginx stub_status moves to the official nginx-prometheus-exporter.
- Keep the 0.2.0 `nchan_*` metric names.

## 0.2.0

- [1](https://github.com/platanus/nchan-prometheus-exporter/pull/1) Add support for shared memory limit metrics from nchan >= 0.1.10

## 0.1.0

* Initial release.