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
#      luci-app-adblock \
#      uhttpd \
#      uhttpd-mod-ubus \
#      luci-base \
#      luci-app-firewall \
#      luci-app-opkg \
#      luci-compat \
#      luci-app-compat \
#      luci-mod-admin-full \
#      luci-mod-rpc
#      luci-app-sqm \
#      luci-app-qos \


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
      luci-app-firewall \
      luci-app-nft-qos \
      luci-compat \
      luci-theme-material \
      luci-app-nlbwmon \
      luci-mod-network \
      luci-mod-rpc \
      luci-mod-status \
      luci-mod-system \
      luci-base \
      luci-compat \
      luci-app-opkg  \
      qrencode



RUN /etc/init.d/uhttpd enable
RUN /etc/init.d/uhttpd restart
RUN /etc/init.d/rpcd enable

COPY backend/controller/ /usr/lib/lua/luci/controller/
COPY backend/view/router-ease /usr/lib/lua/luci/view/router-ease/


RUN opkg list-upgradable | awk '{print $1}' | xargs opkg upgrade || true

RUN echo "iptables -A POSTROUTING -t mangle -p udp --dport 68 -j CHECKSUM --checksum-fill" >> /etc/firewall.user
RUN sed -i '/^exit 0/i cat \/tmp\/resolv.conf > \/etc\/resolv.conf' /etc/rc.local

ARG ts
ARG version
LABEL org.opencontainers.image.created=$ts
LABEL org.opencontainers.image.version=$version

CMD [ "/sbin/init" ]