#!/bin/bash
# Install networking tools
apt-get update
apt-get install -y net-tools iproute2 curl traceroute iputils-ping
# Force all traffic through OpenWRT router
ip route del default 2>/dev/null || true
ip route add default via 192.168.16.2
# Keep init script running to maintain networking configuration
echo "Network configured to use OpenWRT router at 192.168.16.2"
