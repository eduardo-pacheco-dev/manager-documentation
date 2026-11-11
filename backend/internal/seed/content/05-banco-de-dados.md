# Banco de dados e migrações

O banco padrão é **MySQL 8** (ou MariaDB compatível), configurado por **DB_HOST**, **DB_PORT**, **DB_NAME**, **DB_USER** e **DB_PASSWORD**. Também é possível usar **DB_DSN** com uma DSN completa.

## Tabelas

| Tabela | Conteúdo |
| --- | --- |
| users | Contas, permissões e preferências. |
| dev_docs | Páginas da wiki e autor. |
| sessions | Sessões web com token CSRF. |
| api_tokens | Tokens Bearer com hash. |
| notifications | Notificações internas por usuário. |
| settings | Configurações globais (fuso horário). |
| storage_settings | Provedor de upload e credenciais. |

## Migrações

As migrações ficam em **internal/database/database.go** e são aplicadas em ordem com controle em **schema_migrations**. Nunca edite uma migração já aplicada: adicione uma nova lista.

## Backup

Com o servidor em execução, use **mysqldump** para gerar um backup consistente:

```bash
mysqldump --single-transaction -u root -p documentation_center_go > backup.sql
```
