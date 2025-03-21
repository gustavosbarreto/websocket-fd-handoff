#!/bin/sh

/websocket-receiver &
/usr/local/openresty/nginx/sbin/nginx -c /etc/nginx/nginx.conf -g "daemon off;"
