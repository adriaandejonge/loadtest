# Load Test
Generate load with URLs from access log. 

Running a load test is fairly simple: just fire many requests to the server and gradually increase the number. However, the traffic this generates is often different from the patterns actually observed in production. Either a handful URLs is queried again and again or the site is spidered and every URL is accessed an equal number of times. In reality, a site has some popular URLs and some less popular URLs. Both of them are potential sources of latency. By generating load based on an access log, the distribution across URLs is more production-like. Also, typical 404s are tested and some funny query strings may be introduced as they would be in the real world.

Please note:
 * Time stamps are *ignored* The purpose of using an access-log is to get closer to production-like patterns, not replicating the exact traffic.
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
