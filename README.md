# Documentation Center

Reescrita do **Documentation Center** em **Go + React + IBM Carbon Design System**. O projeto mantém os recursos administrativos e a wiki técnica do sistema original e converte todos os plugins em **módulos nativos**, com interface responsiva e suporte a dispositivos móveis.

## Stack

| Camada | Tecnologia |
| --- | --- |
| Backend | Go 1.26+, chi, MySQL |
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
- **Estações**: inventário de 23 campos, página de detalhes, busca/filtros, anexos, comentários, importação/exportação XLSX/CSV e widget de sites.
- **Projetos Nokia**: sites, ROMs, tarefas, checklist de FAMs, datas, interrupções, rejeições, anexos, comentários e widget. Os sites Nokia são sincronizados automaticamente com o cadastro de Estações.
- **Integração entre módulos**: tarefas, follow-ups e eventos da agenda podem ser vinculados a estações e sites Nokia direto das páginas de detalhes, com criação rápida e desvinculação. A **Central de trabalho** no dashboard reúne prazos, follow-ups e compromissos, e as notificações cruzam os módulos com links diretos (follow-up atribuído/atualizado, convites da agenda, rejeições Nokia e lembretes de prazos).
- **Responsividade**: sidebar em gaveta no mobile, grids adaptáveis, tabelas com rolagem e ações empilhadas.
- **Administração inicial**: seed idempotente com administrador e wiki de exemplo.

Todos os plugins (`Agenda`, `FollowUp`, `Nokia`, `Stations` e `Tasks`) foram portados como módulos nativos. O sistema de instalação dinâmica de plugins não existe mais; os módulos fazem parte do binário e podem ser habilitados, ordenados e movidos entre sidebar e menu Módulos nas configurações.

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

- Go 1.26+
- Node.js 20.19+ (recomendado 22+ por causa do Carbon)
- npm
- MySQL 8.0+ (ou MariaDB compatível)

### Banco de dados

Crie o banco e o usuário (o usuário precisa de permissão de `CREATE DATABASE` na primeira execução, ou o banco pode ser criado manualmente):

```sql
CREATE DATABASE documentation_center_go
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

O padrão de desenvolvimento é `DB_USER=root` e `DB_PASSWORD=admin` em `127.0.0.1:3306`. Ajuste em `backend/.env` conforme seu ambiente.

### Backend

```bash
cd backend
cp .env.example .env
go run ./cmd/server
```

Para desenvolvimento com **hot reload** (Air), instale-o uma vez:

```bash
go install github.com/air-verse/air@latest
```

Depois rode a partir da raiz:

```bash
air -c .air.toml
```

Ou de dentro de `backend/`:

```bash
cd backend
air -c .air.toml
```

O Air recompila e reinicia o servidor automaticamente ao alterar arquivos em `backend/` (Go e Markdown dos seeds). Na raiz, o script `run-backend.cmd` garante que o binário rode com working directory em `backend/`, carregando `.env`, banco, uploads e seeds corretamente.

O servidor inicia em `http://localhost:8080`. Na primeira execução:

- conecta ao MySQL configurado e cria o banco `documentation_center_go` se necessário;
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

Ou, a partir da raiz do repositório (atalho para o frontend):

```bash
npm run dev
```

Acesse `http://localhost:5173`. O Vite encaminha `/api` e `/uploads` para o backend em `:8080`.

Outros atalhos na raiz:

```bash
npm run backend    # Air com hot reload
npm run build      # build do frontend
npm run typecheck  # TypeScript
```

### Build de produção

```bash
cd frontend && npm ci && npm run build
cd ../backend
STATIC_DIR=../frontend/dist go run ./cmd/server
```

Ou use Docker (sobe a aplicação e o MySQL):

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
| `DB_HOST` | `127.0.0.1` | Host do MySQL |
| `DB_PORT` | `3306` | Porta do MySQL |
| `DB_NAME` | `documentation_center_go` | Nome do banco |
| `DB_USER` | `root` | Usuário do MySQL |
| `DB_PASSWORD` | `admin` | Senha do MySQL |
| `DB_DSN` | vazio | DSN completa opcional (sobrepõe DB_*) |
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

O workflow [`ci.yml`](.github/workflows/ci.yml) executa automaticamente em pushes na `main` e pull requests:
- backend: `go vet`, build, testes com race detector e cobertura em um serviço MySQL 8;
- frontend: instalação, typecheck e build de produção;
- bundle de produção: compila o binário Linux e valida o frontend compilado.

O workflow [`deploy.yml`](.github/workflows/deploy.yml) só publica depois que o CI termina com sucesso na `main`.

## Deploy na VPS com PM2 e GitHub Actions

O deploy usa **PM2** para manter o binário Go em execução, com releases versionadas e rollback. O GitHub Actions roda os testes, gera o binário Linux (amd64 ou arm64) com o frontend compilado, envia o pacote para a VPS e aciona o `deploy.sh`.

### 1. Preparar a VPS (uma vez)

Requisitos: Ubuntu/Debian ou Rocky/Alma Linux, acesso SSH e um usuário com `sudo`. O setup instala MySQL, Node.js 22 e PM2, configura o início automático no boot, cria os diretórios, provisiona o banco/usuário, gera o `.env` e a senha inicial do administrador.

> Se o `deploy.sh` estiver dentro do repositório (com `.git`), o diretório da aplicação é o próprio repositório. Caso contrário, o padrão é `/opt/documentation-center`.

No servidor atual (`root@russian`, repositório em `~/app/manager-documentation`):

```bash
cd ~/app/manager-documentation
bash deploy.sh setup \
  --url https://docs.exemplo.com \
  --admin-email admin@exemplo.com
```

O setup detecta o diretório do repositório automaticamente. Como o usuário é `root`, o PM2 também rodará como root; para maior isolamento, crie um usuário de deploy e use `--user nome`. Configuração equivalente para o setup genérico:

```bash
curl -fsSLO https://raw.githubusercontent.com/eduardo-pacheco-dev/manager-documentation/main/deploy.sh
sudo bash deploy.sh setup \
  --dir /opt/documentation-center \
  --url https://docs.exemplo.com \
  --admin-email admin@exemplo.com
```

O comando imprime a senha inicial do administrador. Guarde-a. O setup cria o banco e um usuário MySQL dedicado com senha aleatória (salva no `.env`). Se o root do MySQL não usar autenticação por socket, execute com `MYSQL_ROOT_PASSWORD=... sudo -E bash deploy.sh setup ...`. Estrutura criada:

```
<APP_DIR>/
├── current -> releases/<tag>     # release ativa
├── releases/                     # releases (mantém as 5 últimas)
├── shared/.env                   # configuração persistente
├── shared/data/uploads/          # anexos e imagens
├── shared/logs/                  # logs do PM2
└── ecosystem.config.cjs          # configuração do PM2
```

No servidor atual, `<APP_DIR>` é `/root/app/manager-documentation`. O `deploy.sh` mantém `releases/`, `shared/` e `current` dentro do repositório; esses caminhos estão no `.gitignore`.

O `APP_KEY`, as credenciais do MySQL e o `.env` são gerados apenas no setup e preservados nos deploys seguintes. O banco fica no MySQL local da VPS (`DB_NAME`, `DB_USER` e `DB_PASSWORD` no `.env`).

> Use um proxy reverso (Caddy ou Nginx) com HTTPS na frente da porta `8080`. O `PUBLIC_URL` deve ser a URL pública com `https://` para que os cookies seguros sejam ativados.

### 2. Configurar os secrets no GitHub

Em **Settings → Secrets and variables → Actions**:

| Tipo | Nome | Obrigatório | Descrição |
| --- | --- | --- | --- |
| Secret | `VPS_HOST` | sim | IP ou domínio da VPS |
| Secret | `VPS_USER` | sim | Usuário do PM2 (o mesmo `SUDO_USER` do setup) |
| Secret | `VPS_SSH_KEY` | sim | Chave privada SSH de deploy |
| Variable | `VPS_PORT` | não | Porta SSH (padrão `22`) |
| Variable | `VPS_APP_DIR` | não | Diretório da app (padrão `~/app/manager-documentation`; no servidor atual `/root/app/manager-documentation`) |
| Variable | `APP_PORT` | não | Porta HTTP (padrão `8080`) |
| Variable | `APP_BIND` | não | Bind HTTP (padrão `0.0.0.0`) |

A chave pública correspondente deve estar em `~/.ssh/authorized_keys` do usuário `VPS_USER`. O usuário precisa permissão para executar `pm2`. No servidor atual: `VPS_USER=root` e o caminho padrão já resolve para `~/app/manager-documentation`.

### 3. Publicar

Basta fazer merge/push na `main`. O fluxo é:

1. `CI` executa testes e build.
2. `Deploy` compila os pacotes Linux amd64/arm64.
3. O runner identifica a arquitetura da VPS via `uname -m`.
4. O pacote e o `deploy.sh` são enviados por SCP.
5. O `deploy.sh` extrai a release, atualiza o symlink `current`, executa `pm2 startOrReload` e valida `GET /healthz`.
6. Em caso de falha no health check, a release anterior é restaurada automaticamente.

Também é possível disparar manualmente em **Actions → Deploy → Run workflow**.

### 4. Deploy e rollback manuais

```bash
# publicar um pacote já gerado
./deploy.sh --artifact /tmp/documentation-center-linux-amd64.tar.gz --tag manual-001

# voltar para a release anterior
./deploy.sh rollback

# logs e status
pm2 status
pm2 logs documentation-center
```

O `deploy.sh` mantém as 5 releases mais recentes. O rollback troca o symlink `current`, recarrega o PM2 e valida o health check novamente.

### 5. Backup do banco

```bash
mysqldump --single-transaction -u documentation -p documentation_center_go > backup-$(date +%F).sql
```

Guarde também `shared/.env` e `shared/data/uploads/`.


## Observações sobre a migração

- O banco é **MySQL 8** (ou MariaDB compatível), com schema, índices e migrações em `internal/database`. A camada de acesso fica isolada em `internal/store` e `internal/modules`.
- Bancos SQLite antigos não são importados automaticamente. Use as exportações dos módulos (XLSX/CSV) ou uma migração manual de dados ao trocar de banco.
- Os cinco plugins originais (`Agenda`, `FollowUp`, `Nokia`, `Stations` e `Tasks`) foram convertidos em módulos nativos, com tabelas e APIs próprias.
- A importação/exportação dos módulos usa CSV; o módulo Nokia também suporta planilhas XLSX, incluindo modelo, exportação completa e relatório de pendências.
- O armazenamento configurável é compartilhado pelos uploads de imagens e anexos dos módulos.
- A documentação inicial foi reescrita para refletir o novo stack.
