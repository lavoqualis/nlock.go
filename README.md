<!--
SPDX-FileCopyrightText: 2023 KåPI Tvätt AB <peter.magnusson@rikstvatt.se>

SPDX-License-Identifier: MIT
-->

# distributed (by nats.io) locks


# Installation
```shell
go get github.com/lavoqualis/nlock.go
```


# Demo
```shell
docker run -it --rm -p 4222:4222 --name nats-server -v "$($pwd)/contrib:/contrib" nats:latest -c /contrib/test-server.conf


nats kv add --ttl=30s --history=5 --storage=file --max-value-size=1K process_locks
go run ./cmd/

```