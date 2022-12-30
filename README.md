# events

[![Build Status](https://github.com/gerrittrigger/events/workflows/ci/badge.svg?branch=main&event=push)](https://github.com/gerrittrigger/events/actions?query=workflow%3Aci)
[![codecov](https://codecov.io/gh/gerrittrigger/events/branch/main/graph/badge.svg?token=6B10M2ZPPS)](https://codecov.io/gh/gerrittrigger/events)
[![Go Report Card](https://goreportcard.com/badge/github.com/gerrittrigger/events)](https://goreportcard.com/report/github.com/gerrittrigger/events)
[![License](https://img.shields.io/github/license/gerrittrigger/events.svg)](https://github.com/gerrittrigger/events/blob/main/LICENSE)
[![Tag](https://img.shields.io/github/tag/gerrittrigger/events.svg)](https://github.com/gerrittrigger/events/tags)



## Introduction

*events* is the Gerrit events written in Go.



## Prerequisites

- Go >= 1.18.0



## Run

```bash
version=latest make build
./bin/events --config-file="$PWD"/config/config.yml --listen-port=8080
```



## Docker

```bash
version=latest make docker
docker run -v "$PWD"/config:/tmp ghcr.io/gerrittrigger/events:latest --config-file=/tmp/config.yml --listen-port=8080
```



## Usage

```
usage: events --config-file=CONFIG-FILE [<flags>]

gerrit events

Flags:
  --help                     Show context-sensitive help (also try --help-long
                             and --help-man).
  --version                  Show application version.
  --config-file=CONFIG-FILE  Config file (.yml)
  --listen-port=8080         Listen port
  --log-level="INFO"         Log level (DEBUG|INFO|WARN|ERROR)
```



## Settings

*events* parameters can be set in the directory [config](https://github.com/gerrittrigger/events/blob/main/config).

An example of configuration in [config.yml](https://github.com/gerrittrigger/events/blob/main/config/config.yml):

```yaml
apiVersion: v1
kind: events
metadata:
  name: events
spec:
  connect:
    hostname: localhost
    ssh:
      keyfile: /path/to/.ssh/id_rsa
      keyfilePassword: pass
      port: 29418
      username: user
  storage:
    sqlite:
      filename: /path/to/sqlite.db
  watchdog:
    periodSeconds: 20
    timeoutSeconds: 20
```

- spec.connect.hostname: Gerrit host name (e.g., 12:34:56:78)
- spec.watchdog.periodSeconds: Period in seconds (0: turn off)
- spec.watchdog.timeoutSeconds: Timeout in seconds (0: turn off)



## API

- **Request**

```
GET /events/ HTTP/1.0
```



- **Response**

```
HTTP/1.1 200 OK
Content-Disposition: attachment
Content-Type: application/json;charset=UTF-8
{
  "eventBase64": "ZXZlbnRCYXNlNjQ=",
  "eventCreatedOn": 1672214667,
  ...
}
...
```



- **Parameters**

```
since:'TIME': Events after the give 'TIME', in the format 2023-01-01[ 12:34:56].
until:'TIME': Events before the give 'TIME', in the format 2023-01-01[ 12:34:56].
```



- **Examples**

```bash
# Query events which happened between 2023-01-01 and 2023-02-01
curl http://host:port/events/?q=since:2023-01-01+until:2023-02-01
```

```bash
# Query events which happened between 2023-01-01 10:00:00 and 2023-01-01 11:00:00
curl “http://host:port/events/?q=since:%25222023-01-01+10:00:00%2522+until:%25222023-01-01+11:00:00%2522”
```



## License

Project License can be found [here](LICENSE).



## Reference

- [gerrit-events](https://github.com/sonyxperiadev/gerrit-events)
- [gerrit-events-log](https://gerrit.googlesource.com/plugins/events-log/)
- [gerrit-ssh](https://github.com/craftsland/gerrit-ssh)
- [gerrit-ssh](https://gist.github.com/craftslab/2a89da7b65fd32aaf6c598145625e643)
- [gerrit-stream-events](https://gerrit-review.googlesource.com/Documentation/cmd-stream-events.html)
- [gerrit-trigger-playback](https://github.com/jenkinsci/gerrit-trigger-plugin/blob/master/src/main/java/com/sonyericsson/hudson/plugins/gerrit/trigger/playback/GerritMissedEventsPlaybackManager.java)
- [gerrit-trigger-plugin](https://github.com/jenkinsci/gerrit-trigger-plugin)
- [go-queue](https://github.com/alexsergivan/blog-examples/blob/master/queue)
- [go-ssh](https://golang.hotexamples.com/site/file?hash=0x622d73200b734b5b68931b92861d30d6f4ef184f0872a45c49cedf26a29fa965&fullName=main.go&project=aybabtme/multisshtail)
