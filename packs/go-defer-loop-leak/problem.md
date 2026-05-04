# `defer` inside a loop leaks file descriptors

A batch job ingests several thousand small files. Around input number
1000–1024 it dies with:

```
open /var/lib/ingest/0993.json: too many open files
```

`ulimit -n` is the standard 1024 on the host. Read `bug/process.go`
and pick the root cause.

This pattern has caused real production incidents — see
[fleetdm/fleet#42894](https://github.com/fleetdm/fleet/issues/42894)
and [golang/go#68468](https://github.com/golang/go/issues/68468).
