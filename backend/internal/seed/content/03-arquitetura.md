# Arquitetura da aplicação

O projeto é dividido em dois diretórios independentes:

- **backend**: API HTTP em Go, banco MySQL, autenticação, armazenamento e renderização de Markdown.
- **frontend**: SPA React com Carbon Design System, consumindo a API JSON.

## Backend

| Pacote | Responsabilidade |
| --- | --- |
| internal/config | Leitura e validação de variáveis de ambiente. |
| internal/database | Conexão e migrações incrementais do MySQL. |
| internal/store | Repositórios e consultas SQL. |
| internal/models | Entidades e catálogo de módulos/categorias. |
| internal/httpapi | Rotas, middlewares e handlers. |
| internal/widgets | Registro de widgets do dashboard. |
| internal/markdown | Renderização segura de Markdown. |
| internal/notifier | Notificações internas e e-mail. |
| internal/upload | Armazenamento local e S3-compatível. |

## Frontend

A SPA usa React Router, contextos de autenticação/tema/toasts e uma camada **api.ts** centralizada. Os componentes visuais vêm de **@carbon/react**.

## Fluxo de autenticação

1. O navegador envia **POST /api/v1/auth/login**.
2. O servidor valida a senha com bcrypt e cria uma sessão opaca de 7 dias.
3. Um cookie **dc_session** HttpOnly e um token CSRF são emitidos.
4. Requisições mutáveis enviam **X-CSRF-Token**.
5. Clientes de API podem usar **Authorization: Bearer token**.
