# Caller code

```go
if err := db.Lookup(0); err != nil {
    log.Print("oops: ", err)
}
```

# Reproducer

```
$ go run ./cmd/repro
oops: <nil>
```

The log line fires, and the printed error renders as `<nil>` because
the `*DBError` value really is nil — yet `err != nil` evaluates to
true. The `Code: 400` branch is not exercised; `id == 0`.
