Based on your current setup, I'll explain how to build an OpenWrt firmware image you can install on a physical router instead of running it in Docker.

### Step 1: Set up an OpenWrt build environment

```bash
git clone https://github.com/openwrt/openwrt.git
cd openwrt
./scripts/feeds update -a
./scripts/feeds install -a
```

### Step 2: Create a custom config for your router

```bash
make menuconfig
```

In the menu:
- Select your router's target system/subtarget
- Include the packages from your Dockerfile:
    - wpad-wolfssl, curl, iw-full, ip-full, iptables, etc.
- Save your configuration

### Step 3: Copy your custom files into the build

Create a files directory structure matching your target filesystem:

```bash
mkdir -p files/usr/lib/lua/luci/controller/
mkdir -p files/usr/lib/lua/luci/view/router-ease/
cp package/controller/* files/usr/lib/lua/luci/controller/
cp package/view/router-ease/* files/usr/lib/lua/luci/view/router-ease/
```

### Step 4: Build the firmware

```bash
make defconfig
make -j$(nproc) download
make -j$(nproc)
```

### Step 5: Flash the firmware

Your firmware will be in `bin/targets/[architecture]/[subtarget]/`. The exact file to use depends on:

- Your router model
- Flash method (web UI, TFTP, serial)

For example:
- `openwrt-xxx-sysupgrade.bin` - For upgrading from OpenWrt
- `openwrt-xxx-factory.bin` - For flashing from stock firmware

### Alternative approach:

If building proves difficult:

1. Flash standard OpenWrt to your router
2. SSH in and install packages:
   ```bash
   opkg update
   opkg install wpad-wolfssl curl iw-full ip-full iptables ...
   ```
3. Copy your custom LuCI files to the router

Do you know the specific model of router you're targeting?