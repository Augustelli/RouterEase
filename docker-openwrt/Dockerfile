FROM scratch
ADD rootfs.tar.gz /
RUN mkdir -p /var/lock

#RUN opkg remove --force-depends \
#      dnsmasq* \
#      wpad* \
#      iw* && \
#    opkg update && \
#    opkg install luci \
#      wpad-wolfssl \
#      iw-full \
#      ip-full \
#      kmod-mac80211 \
#      dnsmasq-full \
#      iptables-mod-checksum \
#      rpcd \
#      uhttpd \
#      uhttpd-mod-ubus \
#      luci-base \
#      luci-app-firewall \
#      luci-app-opkg \
#      luci-compat \
#      luci-app-compat \
#      luci-mod-admin-full \
#      luci-mod-rpc



RUN opkg remove --force-depends \
      dnsmasq* \
      wpad* \
      iw* && \
    opkg update && \
    opkg install \
      wpad-wolfssl \
      iw-full \
      ip-full \
      cgi-io \
      kmod-mac80211 \
      dnsmasq-full \
      iptables-mod-checksum \
      rpcd \
      uhttpd \
      uhttpd-mod-ubus \
      luci-base \
      luci-mod-rpc \
      libubus-lua \
      libuci-lua \
      luci-lib-jsonc \
      speedtest-netperf \
      luci-app-nlbwmon \
      luci-app-firewall \
      luci-app-sqm \
      luci-app-qos \
      luci-app-nft-qos \
      luci-app-adblock \
      luci-compat \
      luci-theme-material \
      qrencode



RUN /etc/init.d/uhttpd enable
RUN /etc/init.d/uhttpd restart
RUN /etc/init.d/rpcd enable

COPY backend/controller/router-ease.lua /usr/lib/lua/luci/controller/
COPY backend/view/router-ease /usr/lib/lua/luci/view/router-ease/


RUN opkg list-upgradable | awk '{print $1}' | xargs opkg upgrade || true

RUN echo "iptables -A POSTROUTING -t mangle -p udp --dport 68 -j CHECKSUM --checksum-fill" >> /etc/firewall.user
RUN sed -i '/^exit 0/i cat \/tmp\/resolv.conf > \/etc\/resolv.conf' /etc/rc.local

ARG ts
ARG version
LABEL org.opencontainers.image.created=$ts
LABEL org.opencontainers.image.version=$version
LABEL org.opencontainers.image.source=https://github.com/oofnikj/docker-openwrt

CMD [ "/sbin/init" ]