#!/bin/bash
# Install networking tools
apt-get update
apt-get install -y net-tools iproute2 curl traceroute iputils-ping

# Force all traffic through OpenWRT router
ip route del default 2>/dev/null || true
ip route add default via 192.168.16.2

# Explicitly configure DNS to use 192.168.16.2 directly
echo "nameserver 192.168.16.2" > /etc/resolv.conf
echo "search ." >> /etc/resolv.conf
echo "options ndots:0" >> /etc/resolv.conf

# Keep init script running to maintain networking configuration
systemctl restart systemd-resolved 2>/dev/null || true
echo "Network configured to use OpenWRT router at 192.168.16.2 with direct DNS"
