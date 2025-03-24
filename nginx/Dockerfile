FROM openresty/openresty:1.25.3.1-5-alpine-fat

COPY nginx.conf /etc/nginx/nginx.conf
COPY lua /etc/nginx/lua

ENTRYPOINT ["/usr/local/openresty/nginx/sbin/nginx", "-c", "/etc/nginx/nginx.conf"]
