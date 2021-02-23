# nanoRSS
nanoRSS is a simple RSS reader written in Go.

Development is in progress, and the project is not production-ready or useful in any way.

## Available configuration options
* REFRESH_INTERVAL_MINUTES
* DATABASE_DIR
* LOG_REQUESTS

## How to build

Download and install the latest version of Go. Then, run

```
go build
```

# Other versions

nanoRSS contains several abandoned proof-of-concepts to test different data storage libraries (which were discarded):

* [PostgreSQL](../../tree/postgres)
* [Pogreb](../../tree/pogreb)
* [Redis](../../tree/redis)

In addition, there was:
* An early attempt to implement a Bootstrap UI with [native HTML and Javascript](../../tree/nativehtml)
* An old version based on [Bootstrap and jQuery](../../tree/bootstrap)
