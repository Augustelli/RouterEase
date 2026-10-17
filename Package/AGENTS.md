# OpenWrt agents — `Package/`

Read root `AGENTS.md` and `docs/ARCHITECTURE.md`. There are TWO products here: (1) the existing Docker-based OpenWrt emulator for development; (2) an installable router package, **not yet completed in `dev-api`**. Do not confuse them.

- Build RouterEase for an explicitly supported OpenWrt release/architecture with native packaging (`.ipk` for compatible legacy releases or `.apk` for newer releases where appropriate); never infer package manager from device name alone. Pin and document the firmware version and ABI.
- Create a minimal agent controlled by `/etc/init.d/routerease` using `procd`, UCI config `/etc/config/routerease`, and LuCI administration pages. Avoid heavy runtimes. Preserve built-in `dnsmasq`, firewall and network functionality.
- Source of truth for device identity is verified local DHCP leases, neighbor/ARP state and router interface context. Random MAC and IPv6/privacy addressing complicate stable identity; expose mapping confidence and update lifecycle, not false certainties.
- Router API-key setup must be protected, revocable and rotatable; never put secrets in browser JavaScript or logs. Use HTTPS with verified certificates; setup a recoverable enrollment process.
- Before changing UCI DNS/DHCP/firewall/Wi-Fi settings: create backup, validate, apply transactionally, support rollback on connectivity loss. Do not hardcode `192.168.16.0/24`, public backend IPs or production hostnames. Avoid hard-coded `curl` to localhost for backend APIs.
- The lab `run.sh` modifies host network interfaces and firewall; only execute on explicit isolated test machine, never casually on developer laptop. Keep lab-only operations out of a router `.ipk`/`.apk`.
- Ensure UI features correspond to actual implementation. Existing UI speedtest and client enumeration need tests and error handling. Avoid shell injection when generating commands from user/input data.
- Test install/upgrade/uninstall/reboot, service recovery, LuCI permissions, DHCP lease churn, IPv4/IPv6 behavior, allowed/blocked DNS per-device, DNS/backend outages and rollback on a physical reference router.