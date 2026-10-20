#!/bin/sh
# Services read their environment directly; do not render secrets into YAML.
exec "$@"
