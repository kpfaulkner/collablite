# Docker / Kubernetes

One goal is to have the Collablite server run within a Docker/Kubernetes environment.

Currently with the bare-bones Dockerfile, the commands to build and run are:

```
docker build --tag collablite .
docker run  -p 50051:50051 collablite
```

Then the instance can be hit with any existing Collablite client.


