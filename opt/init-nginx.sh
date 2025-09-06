#!/bin/sh

# Install gettext for envsubst
apk add --no-cache gettext

# Generate upstream servers configuration
UPSTREAM_SERVERS=""
OLD_IFS="$IFS"
IFS=','
for server in $GO_API_INSTANCES; do
    if [ -n "$UPSTREAM_SERVERS" ]; then
        UPSTREAM_SERVERS="${UPSTREAM_SERVERS}
        server ${server} max_fails=3 fail_timeout=30s;"
    else
        UPSTREAM_SERVERS="        server ${server} max_fails=3 fail_timeout=30s;"
    fi
done
IFS="$OLD_IFS"

# Replace template variables and generate final nginx.conf
export UPSTREAM_SERVERS
envsubst '${UPSTREAM_SERVERS}' < /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf

# Debug: Show generated configuration
echo "=== Generated NGINX Configuration ==="
cat /etc/nginx/nginx.conf
echo "===================================="

# Test configuration
nginx -t

# Start nginx
exec nginx -g 'daemon off;'
