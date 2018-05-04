#! /bin/bash

set -euo pipefail

serviceName="s2s-test-service"
docker build -t $serviceName .
docker run -e SERVICE_NAME=service1 -e CLUSTER=s2s-test-service -e NODE=s2s-test-service -P $serviceName