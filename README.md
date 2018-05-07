# s2s-test-service
a little service to help test service discovery.

currently this is setup to test connectivity through envoy.  all outbound http requests will be sent to EGRESS_HTTP_PORT with the host header set to a custom value passed as a URL param.

this is meant only to be a small test utility, not something that should be permanently publicly accessible.


```sh
docker run -e SERVICE_NAME=service1 -e SERVICE_PORT=8080 -e EGRESS_HTTP_PORT=9000 chtorr/s2s-test-service:latest
```


```sh
# ping this instance
curl 127.0.0.1:21352/ping

# have the service try to query the db
curl 127.0.0.1:21352/pingdb

# have this service try to ping another using the provided query string as the host header
curl 127.0.0.1:21352/ping_remote?service=service2

# have this service tell another to ping its db, using the provided query string as the host header
curl 127.0.0.1:21352/pingdb_remote?service=service2
```
