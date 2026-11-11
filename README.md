# Documentation Center

Reescrita do **Documentation Center** em **Go + React + IBM Carbon Design System**. O projeto mantém os recursos administrativos e a wiki técnica do sistema original e converte todos os plugins em **módulos nativos**, com interface responsiva e suporte a dispositivos móveis.

## Stack

| Camada | Tecnologia |
| --- | --- |
| Backend | Go 1.24+, chi, SQLite |
| Frontend | React 19, TypeScript, Vite |
| Design system | IBM Carbon Design System (`@carbon/react`) |
| Ícones | Carbon Icons React |
| Tipografia | IBM Plex Sans e IBM Plex Mono |
| Autenticação | Sessão HttpOnly + CSRF e tokens Bearer para API |
| Markdown | Goldmark + Bluemonday |
| Uploads | Disco local ou S3 / Cloudflare R2 |
| API | OpenAPI + Swagger UI |

## Recursos

- **Autenticação**: cadastro, login, logout, sessão segura, CSRF, limite de tentativas e bloqueio de contas inativas.
- **API v1**: cadastro/login com token Bearer, usuário atual, logout e upload de imagens.
- **Dashboard**: widgets de usuários e usuários recentes, modo edição, arrastar para reordenar, redimensionar (1–3 colunas), ocultar e restaurar widgets.
- **Usuários**: busca, filtros por perfil/status, ordenação, paginação, visualização em tabela/cards/lista compacta, criação, edição, ativação/desativação e exclusão. Administradores não podem desativar nem excluir a própria conta.
- **Wiki de desenvolvimento**: categorias, busca, rascunhos/publicados, navegação lateral, Markdown sanitizado, CRUD completo e upload de imagens no editor.
- **Configurações**: tema sistema/claro/escuro, módulos habilitados, ordem e seção de navegação, notificações por e-mail, fuso horário (admin), armazenamento (admin), alteração de senha e exclusão da conta.
- **Notificações**: central com níveis, contador de não lidas, marcar como lida, marcar todas, remover e limpar.
- **Perfil**: dados básicos, papel e data de entrada.
- **Agenda**: calendários próprios e compartilhados, eventos, participantes, recorrências, lembretes, visão mensal/lista e importação/exportação ICS.
- **Tarefas**: listas, tarefas, etapas, Meu Dia, importantes, planejado, recorrência, movimentação e exportação CSV.
- **Follow-ups**: status, prioridade, prazos, responsáveis, clientes/projetos, histórico, configurações, importação/exportação e widget.
- **Estações**: inventário de 23 campos, busca/filtros, anexos, comentários, importação/exportação CSV e widget de sites.
- **Projetos Nokia**: sites, ROMs, tarefas, checklist de FAMs, datas, interrupções, rejeições, anexos, comentários e widget.
- **Exemplo**: demonstração do antigo plugin convertido em módulo.
- **Responsividade**: sidebar em gaveta no mobile, grids adaptáveis, tabelas com rolagem e ações empilhadas.
- **Administração inicial**: seed idempotente com administrador e wiki de exemplo.

Todos os plugins (`Agenda`, `FollowUp`, `Nokia`, `Stations`, `Tasks` e `Example`) foram portados como módulos nativos. O sistema de instalação dinâmica de plugins não existe mais; os módulos fazem parte do binário e podem ser habilitados, ordenados e movidos entre sidebar e menu Módulos nas configurações.

## Estrutura

```
.
├── backend/
│   ├── cmd/server/            # entrada do servidor
│   ├── internal/
│   │   ├── config/            # variáveis de ambiente
│   │   ├── crypto/            # AES-GCM para credenciais
│   │   ├── database/          # conexão e migrações
│   │   ├── httpapi/           # rotas, handlers e OpenAPI
│   │   ├── markdown/          # renderização segura
│   │   ├── models/            # entidades e catálogo
│   │   ├── modules/           # agenda, tarefas, follow-ups, estações e Nokia
│   │   ├── notifier/          # notificações + SMTP
│   │   ├── seed/              # admin, usuários e wiki inicial
│   │   ├── store/             # consultas SQL
│   │   ├── upload/            # local e S3/R2
│   │   └── widgets/           # widgets do dashboard
│   ├── .env.example
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   ├── contexts/
│   │   ├── lib/
│   │   ├── pages/
│   │   └── styles.css
│   ├── package.json
│   └── vite.config.ts
├── docker-compose.yml
└── Dockerfile
```

## Rodando localmente

### Requisitos

- Go 1.24+
- Node.js 20.19+ (recomendado 22+ por causa do Carbon)
- npm

### Backend

```bash
cd backend
cp .env.example .env
go run ./cmd/server
```

Para desenvolvimento com **hot reload** (Air):

```bash
cd backend
go install github.com/air-verse/air@latest
air -c .air.toml
```

O Air recompila e reinicia o servidor automaticamente ao alterar arquivos em `backend/` (Go e Markdown dos seeds). O binário temporário e o banco local ficam em `backend/tmp/` e `backend/data/`, ambos ignorados pelo Git.

O servidor inicia em `http://localhost:8080`. Na primeira execução:

- cria o banco SQLite em `backend/data/app.db`;
- aplica as migrações;
- cria o administrador definido por `SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD`;
- cria o usuário demo `demo@example.com` / `password`;
- publica a wiki inicial.

Credenciais padrão do administrador:

```
E-mail: admin@example.com
Senha:  password
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

Acesse `http://localhost:5173`. O Vite encaminha `/api` e `/uploads` para o backend em `:8080`.

### Build de produção

```bash
cd frontend && npm ci && npm run build
cd ../backend
STATIC_DIR=../frontend/dist go run ./cmd/server
```

Ou use Docker:

```bash
docker compose up --build
```

## Administração inicial

O seed cria um administrador automaticamente. Para criar ou promover outro administrador:

```bash
cd backend
go run ./cmd/createadmin -name "Admin" -email admin2@example.com -password "senha-forte"
```

## Variáveis de ambiente

Consulte [`backend/.env.example`](backend/.env.example). As principais:

| Variável | Padrão | Descrição |
| --- | --- | --- |
| `ADDR` | `:8080` | Endereço HTTP |
| `DATABASE_PATH` | `./data/app.db` | Banco SQLite |
| `UPLOAD_DIR` | `./data/uploads` | Uploads locais |
| `STATIC_DIR` | `../frontend/dist` | Build do frontend |
| `PUBLIC_URL` | `http://localhost:8080` | URL pública usada em links |
| `ALLOWED_ORIGINS` | `http://localhost:5173,...` | Origens CORS |
| `SECURE_COOKIES` | `false` | Cookies apenas HTTPS |
| `SESSION_TTL_HOURS` | `168` | Duração da sessão |
| `APP_KEY` | chave local | Criptografa credenciais de armazenamento |
| `APP_TIMEZONE` | `America/Sao_Paulo` | Fuso usado antes da configuração |
| `SEED_ADMIN_*` | admin@example.com / password | Administrador inicial |
| `SEED_DEMO_DATA` | `true` | Cria demo e wiki inicial |
| `SMTP_*` | vazio | Envio opcional de e-mails |

## API

- Swagger UI: `GET /api/documentation`
- Especificação OpenAPI: `GET /docs`
- Health check: `GET /healthz`

Exemplo de login com token:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password","device_name":"cli"}'
```

Use o token retornado em `Authorization: Bearer <token>`.

> A SPA usa cookie de sessão + `X-CSRF-Token`. Requisições com Bearer token não exigem CSRF.

## Segurança

- Senhas com bcrypt.
- Sessões e tokens persistidos apenas como hash SHA-256.
- Cookies HttpOnly, SameSite=Lax e `Secure` em produção.
- CSRF obrigatório em métodos mutáveis com sessão.
- Limite de 5 tentativas de login/cadastro por minuto por e-mail/IP.
- Markdown com HTML bruto escapado e sanitização por Bluemonday.
- Uploads validados por magic bytes e limite configurável.
- Credenciais de armazenamento criptografadas com AES-GCM.
- Autorização de administrador validada no servidor.

## Testes

```bash
cd backend
go test ./...
go vet ./...

cd ../frontend
npm run typecheck
npm run build
```

Os testes do backend cobrem autenticação, CSRF, usuários, wiki, sanitização de Markdown, widgets, notificações, configurações, tokens de API e upload.

## Observações sobre a migração

- O banco original era MySQL; esta versão usa SQLite por padrão para execução sem dependências externas. A camada de acesso fica isolada em `internal/store` e `internal/modules`.
- Os seis plugins originais (`Agenda`, `FollowUp`, `Nokia`, `Stations`, `Tasks` e `Example`) foram convertidos em módulos nativos, com tabelas e APIs próprias.
- O armazenamento configurável é compartilhado pelos uploads de imagens e anexos dos módulos.
- A documentação inicial foi reescrita para refletir o novo stack.
- A importação/exportação dos módulos usa CSV; o sistema original usava XLSX em alguns fluxos.
