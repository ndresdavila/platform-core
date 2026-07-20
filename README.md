# platform-core

Núcleo de dominio multi-tenant (Go). **Único componente que habla con la base de datos.**

Arquitectura hexagonal (ports & adapters). Pensado para reutilizar el mismo core en varios clientes
(uñas, barbería, spa, clínicas, etc.) vía `tenant_id`.

## Responsabilidad

- Recibe comandos/consultas desde `platform-api` (HTTP interno hoy; gRPC listo para añadir).
- Ejecuta casos de uso de dominio (bookings, pagos, catálogo, clientes).
- Persiste en Postgres a través de adapters (nunca expone SQL hacia fuera).

## No hace

- No conoce Expo, Next ni Keycloak de un cliente concreto.
- No renderiza UI ni tokens de front.

## Layout

```
cmd/server/                 # entrypoint
internal/
  domain/                   # entidades + reglas + ports (interfaces)
  application/              # casos de uso (orquestación)
  adapters/
    http/                   # inbound: API interna
    postgres/               # outbound: repositorios
  platform/                 # wiring (DI manual)
migrations/                 # SQL versionado
```

## Arranque local

```bash
export DATABASE_URL=postgres://user:pass@localhost:5432/platform?sslmode=disable
export HTTP_ADDR=:8083
go run ./cmd/server
```

Health: `GET http://localhost:8083/healthz`

## Contratos internos (ejemplos)

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/v1/tenants/{tenantId}/bookings` | Crear reserva |
| GET | `/v1/tenants/{tenantId}/bookings` | Listar reservas |
| POST | `/v1/tenants/{tenantId}/payments` | Registrar pago |
| GET | `/v1/tenants/{tenantId}/catalog/services` | Catálogo |

Todos los recursos van scoped por `tenantId`.

## Extender a un cliente nuevo

1. Crear tenant: `POST /v1/tenants` `{ "slug": "barber-acme", "name": "Barber Acme" }`.
2. Sembrar catálogo bajo ese `tenantId`.
3. Configurar branding/flags en `platform-api` (no aquí).
4. Apps del cliente hablan solo con `platform-api` + `X-Tenant-Slug`.

Ver también `../PLATFORM.md`.
