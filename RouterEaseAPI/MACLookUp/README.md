# RouterEaseAPI - Documentación del Servicio MACLookup

## Descripción General

El servicio MACLookup es un componente de RouterEaseAPI que permite consultar información sobre fabricantes de dispositivos de red a partir de direcciones MAC. Este servicio facilita la identificación de dispositivos conectados a la red mediante la consulta de bases de datos de OUI (Organizationally Unique Identifier).

## Características

- Consulta de información de fabricantes basada en direcciones MAC
- Soporte para múltiples formatos de entrada de direcciones MAC
- Caché de resultados para mejorar el rendimiento
- Actualización automática de la base de datos de OUI

## API REST

### Endpoint Principal

```
GET /api/v1/mac-lookup/{mac_address}
```

### Parámetros

| Parámetro | Tipo | Descripción | Ejemplo |
|-----------|------|-------------|---------|
| mac_address | string | Dirección MAC a consultar | `00:1A:2B:3C:4D:5E` |

### Formatos de Dirección MAC Soportados

- Con separadores de dos puntos: `00:1A:2B:3C:4D:5E`
- Con separadores de guiones: `00-1A-2B-3C-4D-5E`
- Con separadores de puntos: `001A.2B3C.4D5E`
- Sin separadores: `001A2B3C4D5E`

### Ejemplo de Respuesta Exitosa

```json
{
  "status": "success",
  "data": {
    "mac_address": "00:1A:2B:3C:4D:5E",
    "vendor": "Ejemplo Technologies Inc.",
    "vendor_address": "123 Ejemplo Street, Ciudad Ejemplo, País",
    "vendor_country": "US",
    "block_size": "MA-L",
    "assignment_block": "00:1A:2B",
    "date_created": "2010-01-15",
    "date_updated": "2022-03-20"
  }
}
```

### Ejemplo de Respuesta de Error

```json
{
  "status": "error",
  "message": "Dirección MAC no válida o no encontrada",
  "error_code": "INVALID_MAC"
}
```

## Códigos de Error

| Código | Descripción |
|--------|-------------|
| INVALID_MAC | La dirección MAC proporcionada no tiene un formato válido |
| NOT_FOUND | No se encontró información para la dirección MAC proporcionada |
| DATABASE_ERROR | Error en la base de datos al realizar la consulta |
| RATE_LIMIT_EXCEEDED | Se ha excedido el límite de consultas por minuto |

## Uso desde Go

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/usuario/RouterEaseAPI/maclookup"
)

func main() {
    client := maclookup.NewClient("http://api.ejemplo.com")
    
    info, err := client.Lookup("00:1A:2B:3C:4D:5E")
    if err != nil {
        log.Fatalf("Error en la consulta: %v", err)
    }
    
    fmt.Printf("Fabricante: %s\n", info.Vendor)
    fmt.Printf("País: %s\n", info.VendorCountry)
}
```

## Implementación Interna

El servicio MACLookup utiliza:
- Base de datos IEEE OUI para obtener información de fabricantes
- Sistema de caché para mejorar el rendimiento en consultas repetidas
- Actualizaciones programadas de la base de datos para mantener la información al día

## Límites y Consideraciones

- El servicio tiene un límite de 100 consultas por minuto por IP
- Las actualizaciones de la base de datos OUI se realizan semanalmente
- Algunas direcciones MAC personalizadas o privadas pueden no tener información disponible

## Contribución al Servicio

Si deseas contribuir a mejorar el servicio MACLookup, consulta nuestras [guías de contribución](CONTRIBUTING.md) y considera:

- Añadir soporte para fuentes de datos adicionales
- Mejorar los algoritmos de normalización de direcciones MAC
- Contribuir a la documentación o ejemplos de uso