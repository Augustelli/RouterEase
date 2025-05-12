#!/usr/bin/env bash
docker run -d --name python-client \
  --network openwrt-lan \
  --dns 192.168.16.2 \
  python:alpine /bin/sh -c "
    apk add --no-cache iputils gcc musl-dev python3-dev
    pip install --upgrade pip
    pip install speedtest-cli
    # Set default route via OpenWrt
    ip route del default 2>/dev/null
    ip route add default via 192.168.16.2

    # Test connectivity
    echo 'Python container configured with gateway at 192.168.16.2'
    ping -c 4 192.168.16.2

    # Test speedtest functionality
    echo 'Running initial speedtest...'
    speedtest-cli --simple || echo 'Speedtest failed'

    # Run a simple HTTP server to test connectivity
    echo '<html><body><h1>Python Test Client</h1><p>Connected to OpenWrt router</p></body></html>' > /index.html
    python -m http.server 8080 &

    # Keep container running
    sleep infinity
  "