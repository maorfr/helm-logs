# Helm Logs Plugin

This is a Helm plugin which provides a view of changed Helm releases over time. It works like
`helm history`, but for all releases, sorted by date and has a `--since` option.

## Usage

Print logs of changed Helm releases

```
$ helm logs [flags]
```

### Flags:

```
  -l, --label string              label to select tiller resources by (default "OWNER=TILLER")
      --namespace string          show releases within a specific namespace
      --since duration            Only return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs. (default 1000000h0m0s)
      --storage string            storage type of releases. One of: 'cfgmaps', 'secrets' (default "cfgmaps")
      --tiller-namespace string   namespace of Tiller (default "kube-system")
```


## Install

```
$ helm plugin install https://github.com/maorfr/helm-logs
```

The above will fetch the latest binary release of `helm logs` and install it.

### Developer (From Source) Install

If you would like to handle the build yourself, instead of fetching a binary,
this is how recommend doing it.

First, set up your environment:

- You need to have [Go](http://golang.org) installed. Make sure to set `$GOPATH`
- If you don't have [Glide](http://glide.sh) installed, this will install it into
  `$GOPATH/bin` for you.

Clone this repo into your `$GOPATH`. You can use `go get -d github.com/maorfr/helm-logs`
for that.

```
$ cd $GOPATH/src/github.com/maorfr/helm-logs
$ make bootstrap build
$ SKIP_BIN_INSTALL=1 helm plugin install $GOPATH/src/github.com/maorfr/helm-logs
```

That last command will skip fetching the binary install and use the one you
built.
