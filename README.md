[![Fly Deploy](https://github.com/spwg/golink/actions/workflows/main.yml/badge.svg)](https://github.com/spwg/golink/actions/workflows/main.yml)

# golink

This project is a link shortener that lets you put short keywords into Google
Chrome as a shortcut to some URL. It's a Go server that provides a simple
management interface. To make writing just `go/link` work, right now you need to
start up Google Chrome with a command line flag. The server's set up right now
to run locally on port 10123.

To run the server:

1. `git clone github.com/spwg/golink`
2. `sqlite3 /tmp/golink.db < schema/golink.sql`
3. `cd server`
4. `go run main.go` # runs on port 10123

To run the browser, if you're on MacOS: 
1. Quit Chrome
2. `/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --host-resolver-rules="MAP go localhost:10123"`

To test it out:
1. Open `go/`
2. Create a link "g" with URL "https://google.com"
3. Open a new tab
4. Write `go/g` and hit enter

## Alternatives:

It seems that `chrome.mdns` isn't a supported API for extensions at the moment
per [this issue](https://bugs.chromium.org/p/chromium/issues/detail?id=804945).
Otherwise it would be nice to do a Chrome extension that could make `go/`
resolve to a local server through DNS. That way you wouldn't need to start up a
new Chrome browser.

Another alternative would be to add a line to `/etc/hosts` that makes `go`
resolve to a remote IP hosted by some cloud provider. Then the server could
redirect you to there. The obvious downside of that is that you then have to pay
the latency cost to reach the server, but also then this service would cost
money to run and need to be productionized (security, users and authentication,
etc.).
