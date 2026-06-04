#!/bin/sh
envsubst < /app/config.yaml.template > /app/config.yaml
exec "$@"
