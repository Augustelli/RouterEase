# Ambiente de Desarrollo OpenWrt en Docker
Este proyecto proporciona un ambiente de desarrollo completo basado en OpenWrt ejecutándose en Docker. RouterEasePackage es una pequeña implementación que permite a los desarrolladores probar y desplegar paquetes para routers OpenWrt sin necesidad de hardware dedicado.

Inspirado en otros proyectos que ejecutan hostapd en contenedores, esta solución va un paso más allá al arrancar un sistema operativo de red completo (OpenWrt) que facilita la gestión de todos los aspectos de tu red desde una interfaz web intuitiva.

Para instrucciones específicas de Raspberry Pi, consulta [Construcción en Raspberry Pi](docs/rpi.md).

## Dependencias

* docker
* iw
* iproute2
* envsubst (parte del paquete `gettext` o `gettext-base`)
* dhcpcd

## Construcción
Construye la imagen usando el objetivo `make build`:
```
$ make build
```
Si quieres que paquetes adicionales de OpenWrt estén presentes en la imagen base, añádelos al Dockerfile. De lo contrario, puedes instalarlos con `opkg` después de levantar el contenedor.

Una lista de paquetes con búsqueda está disponible en [openwrt.org](https://openwrt.org/packages/table/start).

## Configuración

La configuración inicial se realiza mediante un archivo de configuración, `openwrt.conf`. Los valores leídos de este archivo en tiempo de ejecución se utilizan para generar archivos de configuración en formato OpenWrt a partir de plantillas en `etc/config/*.tpl`.

Puedes usar el `openwrt.conf.example` incluido como base, que explica los valores.

También es posible hacer cambios persistentes en la interfaz de usuario y descargar una copia de seguridad de toda la configuración de tu router navegando a Sistema > Copia de seguridad / Actualizar firmware y haciendo clic en Copia de seguridad.

## Ejecución

Prepara tu archivo `openwrt.conf` como se explicó anteriormente y ejecuta el objetivo `make run`:
```
$ make run
```

Si llegas a `* Ready`, dirige tu navegador a http://openwrt.home (o lo que hayas configurado en `LAN_DOMAIN`) y deberías ver la página de inicio de sesión. El inicio de sesión predeterminado es `root` con la contraseña establecida como `ROOT_PW`.

Para apagar el router, presiona `Ctrl+C`. Cualquier configuración que hayas realizado o paquetes adicionales que hayas instalado persistirán hasta que ejecutes `make clean`, que eliminará el contenedor.

## Instalación / Desinstalación
```
$ make install
```
Se han incluido objetivos de instalación y desinstalación para `systemd` en el Makefile.

La instalación creará y habilitará un servicio que apunta al directorio donde clonaste este repositorio y ejecutará `run.sh` al arrancar.

## Limpieza
```
$ make clean
```
Esto eliminará el contenedor y todas las redes Docker asociadas para que puedas empezar de nuevo si algo sale mal.

---

## Notas

### Hairpinning

Para que los clientes WLAN se vean entre sí, OpenWrt puentea todas las interfaces en la zona LAN y establece el modo hairpin (también conocido como [reflective relay](https://lwn.net/Articles/347344/)) en la interfaz WLAN, lo que significa que los paquetes que llegan a esa interfaz pueden ser 'reflejados' de vuelta a través de la misma interfaz.

`run.sh` intenta manejar esto si `WIFI_HAIRPIN` está establecido en true, y muestra una advertencia si falla.
El modo hairpin puede no ser necesario en todos los casos, pero si experimentas un problema donde los clientes Wi-Fi no pueden verse entre sí a pesar de que el aislamiento AP está deshabilitado, esto podría solucionarlo.

### Espacio de nombres de red

Para que `hostapd` ejecutándose dentro del contenedor tenga acceso al dispositivo inalámbrico físico, necesitamos establecer el espacio de nombres de red del dispositivo al PID del contenedor en ejecución. Esto hace que la interfaz 'desaparezca' del espacio de nombres de red primario durante la duración del proceso padre del contenedor. `run.sh` comprueba si el host está usando NetworkManager para gestionar la interfaz wifi, e intenta tomarla si es así.

### Actualización

Lee la [guía de actualización](docs/upgrade.md).

---

### Solución de problemas

Los registros se redirigen a `stdout` para que el daemon de Docker pueda procesarlos. Son accesibles con:
```
$ docker logs ${CONTAINER} [-f]
```

Como alternativa a instalar paquetes de depuración dentro de tu router, es posible ejecutar comandos disponibles para el host dentro del espacio de nombres de red. Se crea un enlace simbólico en `/var/run/netns/<container_name>` para mayor comodidad:

```
$ sudo ip netns exec ${CONTAINER} tcpdump -vvi any 
```