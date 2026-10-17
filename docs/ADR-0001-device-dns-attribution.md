# ADR-0001: atribución de dispositivos para DNS

## Estado

Aceptada para el MVP de laboratorio (2026-10-13).

## Decisión

El motor no acepta un `device_id` ni una MAC que entregue un cliente DNS como
prueba de identidad. Un router se autentica con una API key individual, hasheada
con bcrypt y limitada a scopes. Sólo ese router puede renovar leases de
dispositivos ya registrados bajo su `router_id`. Un lease contiene IP, momento
de observación y vencimiento máximo de 900 segundos.

Cada petición DoH usa `X-Router-API-Key` y `X-Router-Client-IP`. La IP se
resuelve contra un lease vigente del mismo router antes de evaluar la política.
Una IP inexistente, expirada o de otro router recibe `403`; no se usa la IP TCP
del backend, que normalmente es la del router/NAT.

El simulador de laboratorio actúa como el router: crea el inventario mediante
la API administrativa y renueva los leases autenticados. Esto demuestra el
contrato y evita la suplantación por un cliente HTTP no autenticado, pero no es
un mecanismo de seguridad de producción: un router comprometido podría mentir
acerca de su propia LAN.

## Camino futuro OpenWrt

El agente/forwarder `procd` debe obtener IP/MAC del cliente desde dnsmasq/ubus,
renovar el lease observado y adjuntar únicamente la IP fuente al enviar la
consulta DoH. La conexión debe usar TLS validado y la API key almacenada sólo
en la configuración protegida del router. Para una versión posterior se
evaluará mTLS por router. MAC aleatorias se tratan como dispositivos distintos;
OUI/MACLookup nunca es una autoridad de identidad.

## Caídas

Las decisiones cacheadas duran cinco minutos y se invalidan inmediatamente
después de cada mutación. Si PostgreSQL no está disponible y no hay una entrada
vigente, la consulta autenticada se reenvía (fail-open explícito) para no dejar
sin DNS a un hogar. Si PowerDNS no responde, se devuelve `SERVFAIL`/HTTP 503;
no se emite una respuesta de bloqueo. Una caída de la API administrativa no
afecta decisiones cacheadas.
