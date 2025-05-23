# RouterEasePackage

## Descripción General

RouterEasePackage es un entorno de desarrollo basado en OpenWrt que se ejecuta en Docker. Permite a los desarrolladores crear, probar y desplegar aplicaciones y paquetes para routers OpenWrt sin necesidad de hardware dedicado.

El proyecto incluye un módulo Lua (`package`) que implementa una pequeña aplicación para la gestión y configuración de routers OpenWrt.


## Estructura del Proyecto

```
RouterEasePackage/
├── Makefile                # Comandos principales para gestionar el entorno
├── build.sh                # Script para construir la imagen Docker
├── run.sh                  # Script para ejecutar el contenedor
├── openwrt.conf            # Archivo de configuración principal
├── openwrt.service         # Archivo de servicio para systemd
└── package/                # Modulo LUA de la aplicación
```

## Requisitos

- Docker
- iw
- iproute2
- envsubst (del paquete `gettext` o `gettext-base`)
- dhcpcd

## Configuración

El archivo `openwrt.conf` contiene toda la configuración necesaria para el entorno:

- Versión de OpenWrt
- Configuración de redes (WAN/LAN)
- Configuración WiFi (si está habilitada)
- Parámetros de contenedor

Revise y modifique este archivo antes de iniciar el entorno.

## Comandos Principales

### Construcción del Entorno

```bash
make build
```

Este comando construye la imagen Docker de OpenWrt con la configuración especificada.

### Ejecución del Entorno

```bash
make run
```

Inicia el contenedor OpenWrt con la configuración definida en `openwrt.conf`.

### Limpieza

```bash
make clean
```

Detiene y elimina el contenedor y las redes Docker asociadas.

### Instalación como Servicio

```bash
sudo make install
```

Instala y configura OpenWrt como un servicio systemd que se iniciará automáticamente al arrancar.

### Desinstalación

```bash
sudo make uninstall
```

Detiene y elimina el servicio systemd.

## Desarrollo del Módulo Lua

El módulo Lua incluido (`package`) proporciona funcionalidades adicionales a OpenWrt. Para desarrollar este módulo:

1. Modifique los archivos dentro de la carpeta `package/`
2. Ejecute `make build` para compilarlo
3. El paquete resultante puede instalarse en OpenWrt mediante `opkg install`

## Acceso a la Interfaz Web

Una vez iniciado el entorno, acceda a la interfaz web de OpenWrt en:

```
http://<IP_CONFIGURADA:192.168.16.2>
```

O use la dirección IP configurada en `LAN_ADDR` (por defecto: 192.168.16.2).

## Notas Técnicas

- La configuración de red usa Docker networks para simular interfaces físicas
- El acceso a dispositivos WiFi requiere configuración adicional
- Si se habilita WiFi, revise la configuración de hairpinning para comunicación entre clientes

## Solución de Problemas

Para ver los registros del contenedor:

```bash
docker logs openwrt_1
```

Para ejecutar comandos de diagnóstico:

```bash
sudo ip netns exec openwrt_1 <comando>
```