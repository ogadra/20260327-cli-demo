#!/bin/sh
set -eu

export NGINX_RESOLVER="${NGINX_RESOLVER:-127.0.0.11}"
export BROKER_HOST="${BROKER_HOST:-broker}"

envsubst '${NGINX_RESOLVER} ${BROKER_HOST}' < /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf

exec nginx -g 'daemon off;'
