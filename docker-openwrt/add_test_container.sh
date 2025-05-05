#!/usr/bin/env bash

docker run -d --name alpine-client \
  --network openwrt-lan \
  --dns 192.168.16.2 \
  alpine:latest /bin/sh -c "
    apk add --no-cache curl iproute2
    # Set default route via OpenWrt
    ip route del default 2>/dev/null
    ip route add default via 192.168.16.2
    # Keep container running
    echo 'Container configured with gateway at 192.168.16.2'
    ping -c 4 192.168.16.2
    sleep infinity
  "
