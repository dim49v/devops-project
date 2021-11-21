#!/bin/sh

cp -f /opt/nginx-confs/default.conf.dist  /etc/nginx/nginx.conf

nginx -g 'daemon off;'