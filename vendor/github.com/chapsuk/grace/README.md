# Grace

[![Build Status](https://travis-ci.org/chapsuk/grace.svg?branch=master)](https://travis-ci.org/chapsuk/grace)
[![GoDoc](https://img.shields.io/badge/GoDoc-Reference-blue.svg
)](https://godoc.org/github.com/chapsuk/grace)
[![codecov](https://codecov.io/gh/chapsuk/grace/branch/master/graph/badge.svg)](https://codecov.io/gh/chapsuk/grace)

Package grace implements set of helper functions around `syscal.Signals`
for gracefully shutdown and reload (micro|macro|nano) services.

```bash
go get -u github.com/chapsuk/grace
```

No breaking changes, follows SemVer strictly.

## Testing

```go
go test -v -cover ./...
```
