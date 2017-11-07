# driver-fitbit implementation notes

implemented in go lang based on the [tplink plug driver](https://github.com/me-box/driver-tplink-smart-plug).

## Development Diary

Add to databox dev environment:
```
./databox-install-component cgreenhalgh/driver-fitbit
```

## Dev build

Now we can (re)build it (in the driver-fitbit directory):
```
docker build -t driver-fitbit -f Dockerfile.dev .
```

From the databox UI (re)start the (empty) driver.

To copy updated files into the container:
```
docker cp . CONTAINERID:/root/go/src/main/
```

Try entering the container (you'll need to find its ID using `docker ps`).
```
docker exec -it CONTAINERID /bin/sh
```

Build and run...
```
GGO_ENABLED=0 GOOS=linux go build -a -tags netgo -installsuffix netgo -ldflags '-d -s -w -extldflags "-static"' -o app src/app.go
./app
```

### Debugging

use `docker ps` to find the container ID for your driver and for the driver's store (image type store-json and name including your driver name).

Check the logs with `docker logs CONTAINERID`

Enter the JSON store with `docker exec -it CONTAINERID /bin/sh` and check the database directly:
```
mongo
use db
db.timesseries.find()
db.KV.find()
```
