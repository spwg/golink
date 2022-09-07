[![Fly Deploy](https://github.com/spwg/golink/actions/workflows/main.yml/badge.svg)](https://github.com/spwg/golink/actions/workflows/main.yml)

# golink

This project is a link shortener that lets you put short keywords into Google
Chrome as a shortcut to some URL. It's a Go server that provides a simple
management interface. To make writing just `go/link` work, right now you need to
change /etc/hosts.

To run the server:

1. `git clone github.com/spwg/golink`
2. `go run main.go` # runs on port 10123

To use the version in prod:

1. Add an entry to `/etc/hosts`:

```shell
$ sudo echo "2a09:8280:1::1:66ae	go # golinkservice.com" >> /etc/hosts
```

2. Open `go/`
3. Create a link "g" with URL "https://google.com"
4. Open a new tab
5. Write `go/g` and hit enter

To use a local version, it's easiest to just go to the server directly because 
otherwise you need to startup Google Chrome from the command line.

## Alternatives:

It seems that `chrome.mdns` isn't a supported Google Chrome extension API at the moment
per [this issue](https://bugs.chromium.org/p/chromium/issues/detail?id=804945).
Otherwise it would be nice to do a Chrome extension that could make `go/`
resolve to a local server through DNS. That way you wouldn't need to change /etc/hosts.
