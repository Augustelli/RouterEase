FROM scratch
ADD rootfs.tar.gz /
RUN mkdir -p /var/lock


RUN opkg update
RUN opkg remove --force-depends \
      dnsmasq* \
      wpad* \
      iw* && \
    opkg install \
      wpad-wolfssl \
      curl \
      iw-full \
      ip-full \
      iptables \
      iptables-mod-extra \
      ip6tables \
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
      luci-theme-openwrt-2020 \
      luci-mod-network \
      luci-mod-rpc \
      luci-mod-status \
      luci-mod-system \
      luci-base \
      luci-compat \
      luci-app-opkg  \
      luci-app-nlbwmon \
      nlbwmon \
      luci-app-adblock-fast \
      python3-speedtest-cli \
      arp-scan \
      qrencode



RUN /etc/init.d/uhttpd enable
RUN /etc/init.d/uhttpd restart
RUN /etc/init.d/rpcd enable
RUN service nl

COPY backend/controller/ /usr/lib/lua/luci/controller/
COPY backend/view/router-ease /usr/lib/lua/luci/view/router-ease/


RUN opkg list-upgradable | awk '{print $1}' | xargs opkg upgrade || true

RUN echo "iptables -A POSTROUTING -t mangle -p udp --dport 68 -j CHECKSUM --checksum-fill" >> /etc/firewall.user
RUN sed -i '/^exit 0/i cat \/tmp\/resolv.conf > \/etc\/resolv.conf' /etc/rc.local

RUN mkdir -p /usr/share/arp-scan /etc/arp-scan && \
    wget -O /usr/share/arp-scan/ieee-oui.txt https://raw.githubusercontent.com/royhills/arp-scan/master/ieee-oui.txt || \
    curl -o /usr/share/arp-scan/ieee-oui.txt https://raw.githubusercontent.com/royhills/arp-scan/master/ieee-oui.txt && \
    cp /usr/share/arp-scan/ieee-oui.txt /etc/arp-scan/mac-vendor.txt



ARG ts
ARG version
LABEL org.opencontainers.image.created=$ts
LABEL org.opencontainers.image.version=$version

CMD [ "/sbin/init" ]