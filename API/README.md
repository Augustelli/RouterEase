# RouterEaseAPI

## Descripción General

RouterEaseAPI es un conjunto de servicios y herramientas desarrollados en Go que facilitan la gestión, monitorización y configuración de dispositivos de red, especialmente routers basados en OpenWrt. Este proyecto proporciona APIs RESTful que pueden ser consumidas por aplicaciones cliente o interfaces web.



## Estructura del Proyecto

```
RouterEaseAPI/
├── MACLookUp/          # Servicio de consulta MAC
└── README.md           # Este archivo
```


## Componentes Principales

### MACLookup

Servicio para consultar información sobre fabricantes de dispositivos a partir de direcciones MAC. Permite identificar dispositivos conectados a la red mediante la consulta de bases de datos OUI (Organizationally Unique Identifier).

[Documentación completa de MACLookup](MACLookUp/README.md)



