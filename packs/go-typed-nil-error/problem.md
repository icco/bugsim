# Typed-nil error sneaks past `err != nil`

A teammate adds a thin lookup wrapper around a database client. The
unit test asserts that "successful lookups return a nil error". The
test fails — the caller-side log line `oops: <nil>` keeps appearing —
even though the function appears to return `nil`.

Read `bug/lookup.go` and pick the answer that best explains why.

The Go FAQ entry
[Why is my nil error value not equal to nil?](https://go.dev/doc/faq#nil_error)
covers the underlying mechanic; in production this pattern has been
called the most-stepped-on rake in the language.
