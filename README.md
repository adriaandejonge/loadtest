# Load run
Run load with URLs from access log. 

Please note:
 * Time stamps are *ignored* The purpose of using an access-log is to get closer to production-like patterns.
 * Only GET requests (no POST/PUT/HEAD/DELETE/etc)
 * Status codes in the access log are ignored so 200, 301, 302, 404, 500 etc are still getting called
 * Any query strings are still in the URL

## Running:

```
./loadrun <bigswitch-log-file-name> <number-of-concurrent-requests>
```


e.g.:
```
./loadrun ./access_log-2018-09-16-1537084023 3
```

## Building

Make sure you have a recent version of Go installed.

```
GOOS=linux GOARCH=amd64 go get github.com/adriaandejonge/loadrun
GOOS=windows GOARCH=amd64 go get github.com/adriaandejonge/loadrun
GOOS=darwin GOARCH=amd64 go get github.com/adriaandejonge/loadrun
```

You will probably want top edit the `BASEURL` constant after the first build. Change this in `~/go/src/github.com/adriaandejonge/loadrun` and build again
