# Pruebas Postman — DNS Policy

La colección Git-native está en `postman/collections/routerease-dns-policy` y
el entorno sin secretos en `postman/environments/routerease-lab.environment.yaml`.
Importa esa carpeta desde Postman Desktop, selecciona **RouterEase lab** y carga
un access token Keycloak válido como valor secreto de `admin_access_token`.

El token debe tener audiencia `routerease-api` y rol de realm
`routerease-admin`. En un realm de laboratorio, habilita Direct Access Grants
para un cliente de pruebas o usa OAuth 2.0 Authorization Code + PKCE de
Postman. No guardes usuario, contraseña, token ni API key en Git.

Con el Compose normal, usa `base_url=http://127.0.0.1/routerease/dns` y deja
`routerease_host=dev.routerease.com.ar`; la colección añade el Host requerido
por Traefik. Si expones Go directamente en un laboratorio, cambia `base_url`
a `http://127.0.0.1:8080` y deja `routerease_host` vacío.

Ejecuta las 15 solicitudes en orden. Capturan IDs y la API key sólo en el
runtime de la colección. Las aserciones prueban CRUD, membresía, regla BLOCK,
política efectiva BLOCK, renovación autenticada de leases, actualización a
ALLOW sin reinicio, y rechazo de una key revocada.

Postman puede enviar `application/dns-message` como archivo, pero no decodifica
de forma nativa el RCODE DNS binario. Para la evidencia A/B real —misma consulta,
niño NXDOMAIN y adulto permitido— ejecuta la prueba Go integrada descrita en
`docs/OPERATIONS-DNS-POLICY.md`; ésta desempaqueta el mensaje DNS y verifica el
ID y RCODE. La colección cubre el contrato HTTP y la prueba Go cubre el wire
format DNS.

Validación local de los artefactos:

```sh
postman collection lint postman/collections/routerease-dns-policy
postman environment lint postman/environments/routerease-lab.environment.yaml --fail-severity warning
postman collection run postman/collections/routerease-dns-policy \
  -e postman/environments/routerease-lab.environment.yaml
```

El último comando requiere que el secreto `admin_access_token` sea resoluble
para la CLI; Postman Desktop es la opción más simple para la primera ejecución.
