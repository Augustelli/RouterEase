#!/usr/bin/env bash
cat > init-script.sh << 'EOF'
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
EOF

# Make it executable
chmod +x init-script.sh

# Now run the container with the init script mounted as a startup script
docker run -d \
  --name webtop-client \
  --network openwrt-lan \
  --dns 192.168.16.2 \
  --mac-address=d0:f4:05:21:cb:aa \
  --cap-add NET_ADMIN \
  -p 43000:3000 \
  -e PUID=1000 \
  -e PGID=1000 \
  -e TZ=UTC \
  -v $(pwd)/init-script.sh:/custom-init.sh \
  -e S6_CMD_WAIT_FOR_SERVICES_MAXTIME=0 \
  -e S6_CMD_WAIT_FOR_SERVICES=1 \
  lscr.io/linuxserver/webtop:ubuntu-xfce

# Execute the init script in the running container
docker exec webtop-client /bin/bash /custom-init.sh