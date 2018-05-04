FROM golang:1.9.4-alpine3.7 as builder
# RUN mkdir /app 
ADD . /go/src/app
WORKDIR /go/src/app
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s" -installsuffix cgo -o bin/app .

FROM chtorr/envoy-docker-base:v1.6.0
EXPOSE 8080
WORKDIR /
COPY --from=builder /go/src/app/bin/app .
