# distributed (by nats.io) locks

```shell
docker run -it --rm -p 4222:4222 --name nats-server -v "$($pwd)/contrib:/contrib" nats:latest -c /contrib/test-server.conf


nats kv add --ttl=30s --history=5 --storage=file --max-value-size=1K process_locks
go run ./cmd/

```