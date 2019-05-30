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

- You need to have [Go](http://golang.org) >= 1.11 installed (we use `go modules`)

Clone the repo (not necessarily into your `$GOPATH`). You can use `go get -d github.com/maorfr/helm-logs` if you plan to work inside `$GOPATH`.
for that.

```
$ cd <workdir>/github.com/maorfr/helm-logs
$ make bootstrap build
$ SKIP_BIN_INSTALL=1 helm plugin install <workdir>/github.com/maorfr/helm-logs
```

That last command will skip fetching the binary install and use the one you
built.
