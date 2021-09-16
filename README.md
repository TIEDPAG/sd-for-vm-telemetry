# Service Discovery for VM Telemetry
## Overview
This repo contains an experimental feature to support VM
telemetry with [file-based service discovery](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#file_sd_config) of Prometheus.
For more information, one could access the RFC [here](https://docs.google.com/document/d/1gP12rR2fUV0iHpABiwFiQSy-M1AfyR2XDbQvQil64-M/edit?usp=sharing).
This repo provides the code to build the binary, which is
hosted on [Docker hub](https://hub.docker.com/repository/docker/jackyzz/vm-discovery). 
The container will be deployed along with Prometheus as
a sidecar, which will watch for
the updates to the workload entries registered with VMs, and
write the endpoint IP to a config map. The config map will then
be mounted by the Prometheus as file, thus the service discovery.
A sample of Prometheus deployment could be found in `samples/prometheus.yaml`.

## Usage
To build the binary, simply run:
```
make build
```
The binary will be written to `out` directory. 

To build the docker image, simply run:
```
make docker
```

To build the docker image and push, update the `DOCKER_REPO`
and run:
```
make docker.push
```

## Update
In the original plan, during the istio version upgrade process, the new fields in the proto file will cause the watch to return an error, which will cause the application to crash, and the community-provided plan is to increase the container operation in the promethus pod. This error will cause promethus It is unable to enter the healthy state, so fork has made a response modification. The modification points are as follows:

- v0.2.0
  - Support istio 1.10.3
  - Retry when watch returns an error
- v0.3.0
  - Use file save sd config
  - Namespace isolation watcher
  - Use label enabled watcher