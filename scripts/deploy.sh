#!/usr/bin/env bash
set -e
set -o pipefail

GIT_COMMIT=$(git rev-parse HEAD)
SERVICE_NAME=ipapi
DOCKER_IMAGE=ghcr.io/alex-oleshkevich/ipapi:$GIT_COMMIT
PACKAGE="github.com/alex-oleshkevich/ipapi"
BUILD_TIMESTAMP=$(date '+%Y-%m-%dT%H:%M:%S')
LDFLAGS=(
  "-X '${PACKAGE}/pkg/version.CommitHash=${COMMIT_HASH}'"
  "-X '${PACKAGE}/pkg/version.BuildTime=${BUILD_TIMESTAMP}'"
)

docker build --build-arg=GIT_COMMIT=$GIT_COMMIT -t $SERVICE_NAME -t $DOCKER_IMAGE .
docker push $DOCKER_IMAGE

ssh alex@ssh.aresa.me docker pull $DOCKER_IMAGE
ssh alex@ssh.aresa.me docker volume create $SERVICE_NAME
ssh alex@ssh.aresa.me docker service rm $SERVICE_NAME || true
ssh alex@ssh.aresa.me docker service create \
    --name $SERVICE_NAME \
    --replicas 1 \
    --update-delay 10s \
    --update-parallelism 1 \
    --update-monitor 10s \
    --update-order start-first \
    --restart-condition any \
    --network traefik \
    --with-registry-auth \
    --env GEOIP_DB_PATH=/data/GeoLite2-City.mmdb \
    --label traefik.enable=true \
    --label 'traefik.http.routers.ipapi.rule="Host (\`ip.aresa.me\`)"' \
    --label traefik.http.routers.$SERVICE_NAME.tls.certResolver=letsencrypt \
    --label traefik.http.routers.$SERVICE_NAME.service=$SERVICE_NAME \
    --label traefik.http.services.$SERVICE_NAME.loadbalancer.server.port=8080 \
    "$DOCKER_IMAGE"
