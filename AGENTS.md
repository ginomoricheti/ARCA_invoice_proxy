# AGENTS.md — ARCA Invoice Proxy

Instrucciones operativas para agentes que trabajen sobre este repositorio.
Este documento integra el sistema de trabajo Git centralizado (`git-management.ps1`) con las convenciones propias del proyecto.

---

## 1. Identidad del repositorio

**ARCA Invoice Proxy** es una API backend en **Go** que abstrae la facturación electrónica ante ARCA (ex AFIP) en Argentina.

- **Módulo**: `arca-invoice-proxy` (`go.mod`), **Go 1.23**.
- **Base de datos**: PostgreSQL, driver `pgx/v5` + `pgxpool`.
- **Arquitectura**: Clean Architecture, dependencies apuntan hacia adentro:
  - `cmd/api/` → entry point (`main.go`).
  - `internal/domain/` → reglas de negocio puras, **cero dependencias externas** (`invoice`, `customer`, `credential`, `apikey`, `idempotency`, `errors`).
  - `internal/application/` → use cases (`invoice`, `authentication`, `idempotency`); define interfaces que implementa la infraestructura.
  - `internal/infrastructure/` → adaptadores externos (`postgres`).
  - `internal/interfaces/http/` → handlers, middleware y DTOs.
  - `migrations/` → SQL de esquema (`001_initial_schema.sql`).
  - `tests/` → archivos `*_test.go` de dominio y aplicación.
  - `docs/` → `architecture.md`, `api.md`, `security.md`, `database.md`, `openapi.yaml`.
- **Config**: variables de entorno (ver `.env.example`): `PORT`, `DATABASE_URL`, `API_KEY_PEPPER`, `ARCA_*`, `IDEMPOTENCY_TTL`, `LOG_*`. No committear `.env`.

### Convenciones de código del proyecto

1. **Ley de dependencias**: `interfaces → application → domain ← infrastructure`. Nunca romper el flujo hacia adentro; `domain` no debe importar paquetes externos, de infraestructura ni de HTTP.
2. **Tipado fuerte**: usar tipos propios (`InvoiceType`, `IVACondition`, `Environment`, `IdempotencyStatus`) en lugar de `string` sueltos, con constructores que validan.
3. **Errores**: usar `AppError` de `internal/domain/errors` con códigos (`CodeInvalidCUIT`, `CodeIdempotencyConflict`, `CodeARCARejected`, etc.) y `Wrap` con contexto. Nunca exponer stack traces ni detalles internos en respuestas.
4. **Repositorios**: interfaces pequeñas y específicas (`InvoiceRepository`, `UserRepository`, `ARCAClient`, `IdempotencyStore`). La infraestructura las implementa; el dominio no conoce PostgreSQL.
5. **SQL siempre parametrizado**; idempotencia con request hash (SHA-256) y estados `processing | succeeded | failed` con TTL.
6. **API keys**: nunca en texto plano; hash HMAC-SHA256 + pepper. El prefijo distingue entorno (`sk_live_` vs `sk_test_`).
7. **ARCA**: en el repo hay solo un `ARCAClient` mock (MVP). La integración real WSAA/WSFEv1 es fase 2. No asumir que está implementada.
8. **Tests**: residen en `tests/` (paquete a nivel repo raíz). Todo código nuevo debe tener tests equivalentes.

---

## 2. Sistema de ramas

```text
dev  = rama EXCLUSIVA de desarrollo
main = rama estable/producción (protegida)
```

El agente:

- trabaja **exclusivamente** en `dev`;
- verifica la rama con `git branch --show-current` **antes** de modificar código;
- **nunca** desarrolla directamente sobre `main`;
- **nunca** hace merge de `dev` hacia `main`;
- **nunca** hace push a `main`.

Prohibido explícitamente:

```bash
git push origin main
git push --force
git push --force-with-lease
```

`main` es responsabilidad del usuario. Si no existe `main` local o remoto, **no crearla el agente**.

---

## 3. Sistema Git centralizado (`git-management.ps1`)

El script de gestión vive fuera de este repositorio, en la raíz del directorio `dev/`:

```text
C:\Users\ginom\Documents\GitHub\dev\git-management.ps1
```

Desde la raíz de este proyecto (`C:\Users\ginom\Documents\GitHub\dev\ARCA Invoice Proxy`), la ruta relativa es:

```powershell
..\git-management.ps1
```

Cuando esté disponible, es la **interfaz principal** para operaciones Git. Operaciones:

```powershell
..\git-management.ps1 status
..\git-management.ps1 setup
..\git-management.ps1 validate
..\git-management.ps1 sync
..\git-management.ps1 commit "tipo: descripción"
..\git-management.ps1 push
```

- `status` → estado de todos los proyectos bajo `dev/`.
- `setup` → asegura rama `dev` e instala el hook `commit-msg`.
- `validate` → validaciones previas a commit (ver §4).
- `sync` → `git pull --ff-only origin dev`.
- `commit "tipo: desc"` → valida el mensaje, corre validación y commitea.
- `push` → push únicamente a `origin/dev`.

> El script opera sobre **todos** los repositorios bajo `dev/`. Si un cambio pertenece solo a este proyecto, confirmar que el commit/push se genera acá (el script itera proyectos; los que no tienen cambios se saltan con "No hay cambios").

---

## 4. Validación obligatoria antes de commit

Antes de cualquier commit ejecutar:

```powershell
..\git-management.ps1 validate
```

Para este proyecto (Go), `validate` ejecuta:

- **gofmt** (`gofmt -l .` — no deben existir archivos sin formatear; si aparecen, correr `gofmt -w .`);
- **go vet** (`go vet ./...`);
- **go test** (`go test ./...`).

Requisito: Go 1.23+ disponible en PATH.

Una tarea **no está terminada** si las validaciones relevantes fallan. No ocultar errores ni borrar tests para lograr una validación exitosa. Si la validación falla, corregir el código y revalidar.

---

## 5. Conventional Commits (obligatorio)

Formato: `tipo(scope?): descripción` o `tipo!:` / `tipo(scope)!:` para breaking changes.

Tipos permitidos:

```text
feat  fix  refactor  test  docs  chore  build  ci
```

Ejemplos válidos:

```text
feat: add invoice parser
fix: handle malformed PDF
fix(pdf): handle invalid statement
refactor: simplify transaction parser
test: add parser tests
docs: update README
chore: update dependencies
build: update build configuration
ci: add GitHub Actions workflow
feat!: change API
fix(api)!: change response format
```

Mensajes genéricos **prohibidos**:

```text
arreglos
cambios
update
fix
cosas
terminado
```

Existe un hook `commit-msg` (instalado por `..\git-management.ps1 setup`) que valida esta regla técnicamente y **rechaza** los commits que no la cumplan. El agente **nunca** debe eliminar, desactivar ni modificar el hook para saltarse la validación.

---

## 6. Flujo obligatorio del agente

```text
1.  Verificar rama (debe ser dev)
2.  Inspeccionar estado (git status / ..\git-management.ps1 status)
3.  Analizar código existente (estructura, convenciones, docs)
4.  Implementar cambios
5.  Ejecutar validaciones (..\git-management.ps1 validate)
6.  Revisar git diff antes de commitear
7.  Crear Conventional Commit
8.  Push únicamente a origin/dev (..\git-management.ps1 push)
9.  Informar resultado
```

**El agente no hace commits antes de validar.**

---

## 7. Protección de cambios del usuario

El agente **nunca** ejecuta automáticamente, para resolver cambios existentes:

```bash
git reset --hard
git clean -fd
git stash
```

Si encuentra modificaciones que no realizó:

```bash
git status
git diff
```

Debe **preservarlas**. Si interfieren con la tarea, informar el problema al usuario **antes** de cualquier operación potencialmente destructiva.

---

## 8. Secretos

Nunca committear:

```text
.env
.env.*
*.pem
*.key
credentials.json
secrets.json
```

salvo archivos de ejemplo explícitos como `.env.example` (que en este repo es la plantilla permitida).

Nunca incluir en código ni commits:

- API keys
- tokens
- passwords
- credenciales
- secretos

En particular: `API_KEY_PEPPER`, certificados ARCA (`ARCA_CERT_PATH`/`ARCA_KEY_PATH`) y el valor de `DATABASE_URL` real no deben aparecer en el repositorio. Las claves API se manejan hasheadas; nunca persistir valores planos.

---

## 9. Alcance

Realizar únicamente los cambios necesarios para la tarea solicitada. No ejecutar automáticamente:

- refactors masivos
- cambios de arquitectura
- actualizaciones generales de dependencias
- cambios de estilo globales
- migraciones no solicitadas
- modificaciones de archivos no relacionados

Si se detectan mejoras potenciales fuera del alcance, **informarlas sin implementarlas**.