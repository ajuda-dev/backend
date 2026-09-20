# AjudaDev Backend

API da plataforma **AjudaDev**: comunidades, eventos, skills e usuários. O backend autentica com JWT (cookie HttpOnly `ajudadev_session` no navegador, ou `Authorization: Bearer` em clientes nativos/API).

## O que é

Serviço REST em **Go 1.24** (Fiber + GORM) para:

- cadastro e login de usuários (e-mail/senha ou OAuth GitHub)
- endereços com validação via **ViaCEP**
- comunidades (dono + endereço), membros e listagem paginada
- eventos (webinar, mentoring, community event), participação e convites
- catálogo de skills e associação a usuários
- notificações (inbox + SSE) e verificação de e-mail

A API sobe na porta **8080**. Documentação interativa em `/swagger` (quando `SWAGGER_ENABLED` não for `false`). Health check público: `GET /health`.

## Banco de dados

**PostgreSQL**. A conexão é feita no boot (`GORM` + driver Postgres). O schema é criado/atualizado com `AutoMigrate`; a extensão `unaccent` é habilitada automaticamente (usada nas buscas por nome/cidade).

Variáveis de conexão:

| Variável | Uso |
|---|---|
| `DB_HOST` | Host e porta, no formato `host:porta` (ex.: `localhost:5432`). Sem porta, assume `5432`. |
| `DB_USER` | Usuário |
| `DB_PASSWORD` | Senha |
| `DB_NAME` | Nome do banco |
| `DB_SSL_MODE` | SSL (`disable` em local) |
| `DB_TIME_ZONE` | Timezone da sessão (ex.: `America/Sao_Paulo`) |

`DB_PORT` no `.env.exemple` **não é lido**; a porta vai em `DB_HOST`.

## Como usar

Pré-requisitos: **Go 1.24+** e um **PostgreSQL** acessível. Docker é necessário só para os testes de integração (Testcontainers).

1. Copie o template e preencha os valores:

```bash
cp .env.exemple .env
```

2. Suba a API:

```bash
go run ./cmd/api
```

A API escuta em `http://localhost:8080`.

3. (Opcional) Rotina de purga física, em processo separado:

```bash
go run ./cmd/purge
```

Sem `PURGE_ENABLED=true`, o processo registra que o job está desabilitado e encerra.

Outros comandos:

```bash
go build ./...
go test ./...
```

`go test ./...` sobe um Postgres 15 via Testcontainers (precisa de Docker). Swagger: `swag init` gera `docs/`.

Imagem Docker: `Dockerfile` (API + binário de purge). O Compose da stack fica no repositório de deploy (`ajudadev/deploy`).

## Variáveis de ambiente

Template: `.env.exemple`. Carregadas com `godotenv` no boot.

### Obrigatórias (sempre)

Sem estas a API não sobe de forma útil:

| Variável | Motivo |
|---|---|
| `JWT_SECRET` | Obrigatório no boot, inclusive em development. Sem ele a API aborta. |
| `DB_HOST` | Conexão com o Postgres |
| `DB_USER` | Conexão com o Postgres |
| `DB_PASSWORD` | Conexão com o Postgres |
| `DB_NAME` | Conexão com o Postgres |

`DB_SSL_MODE` e `DB_TIME_ZONE` devem ir no `.env` (não têm default no código; o template usa `DB_SSL_MODE=disable`).

### Obrigatórias em production (`APP_ENV=production`)

O boot é fail-closed. Além das anteriores:

| Variável | Regra |
|---|---|
| `EMAIL_CODE_SECRET` | Obrigatório e **diferente** de `JWT_SECRET` (códigos de e-mail / reset). |
| `SESSION_COOKIE_SECURE` | Deve ser `true` |
| `OAUTH_COOKIE_SECURE` | Deve ser `true` |

Em **development** (`APP_ENV` vazio ou qualquer valor que não seja `production`):

- `EMAIL_CODE_SECRET` vazio cai no `JWT_SECRET` (com warn no boot)
- `SESSION_COOKIE_SECURE=false` e `OAUTH_COOKIE_SECURE=false` permitem HTTP local

### Opcionais (têm default ou desligam o recurso)

| Variável | Default / comportamento |
|---|---|
| `APP_ENV` | `development` se vazio. Só `production` ativa o fail-closed. |
| `JWT_EXPIRATION_TIME` | Horas de validade do JWT (template: `24`) |
| `SWAGGER_ENABLED` | `true` se vazio; `false` omite `/swagger` |
| `LOG_OUTPUT` | `stdout` (arquivo se for um path) |
| `LOG_LEVEL` | `info` |
| `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` | Sem os dois, OAuth GitHub fica desligado |
| `GITHUB_CALLBACK_URL` | `http://127.0.0.1:8080/v1/auth/github/callback` |
| `OAUTH_FRONTEND_URL` | URL do front após o callback (template: `http://localhost:3000`) |
| `EMAIL_PROVIDER` | `brevo` |
| `BREVO_SMTP_USER` / `BREVO_SMTP_KEY` / `BREVO_FROM` | Sem os três, e-mail vira noop (dev/CI sem SMTP) |
| `BREVO_SMTP_HOST` / `BREVO_SMTP_PORT` | `smtp-relay.brevo.com` / `587` no template |
| `EMAIL_CODE_TTL_MINUTES` | `5` |
| `EMAIL_CODE_RESEND_INTERVAL_SECONDS` | `60` |
| `EMAIL_CODE_MAX_SENDS_PER_HOUR` | `5` |
| `EMAIL_CODE_MAX_ATTEMPTS` | `5` |
| `OUTBOX_ENABLED` | `true` |
| `OUTBOX_POLL_INTERVAL_MS` | `2000` |
| `OUTBOX_BATCH_SIZE` | `50` |
| `PURGE_ENABLED` | `false` |
| `PURGE_OLDER_THAN_DAYS` | `30` |
| `PURGE_INTERVAL_HOURS` | `24` |

Quotas e rate limits (`MAX_OWNED_COMMUNITIES`, `MAX_CREATED_SKILLS`, `RATE_LIMIT_*`, etc.) também são opcionais: o template traz os defaults. `ADMIN` ignora quotas e rates; `MODERATOR` usa as vars `*_MODERATOR`. `MAX_SKILLS_PER_USER` vazio = sem limite.

## Autenticação (resumo)

Quase todos os endpoints exigem JWT. Exceções públicas: `POST /v1/user/register`, `POST /v1/user/login`, `POST /v1/user/logout`, `POST /v1/user/forgot-password`, `POST /v1/user/reset-password`, `GET /health` e `/swagger/*`.

- Navegador: cookie HttpOnly `ajudadev_session` (login, registro e callback OAuth).
- Cliente nativo: header `X-Client-Type: native` no login/registro devolve `token` no body; use `Authorization: Bearer <jwt>`.
