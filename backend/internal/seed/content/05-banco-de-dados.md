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

O sistema possui um módulo nativo de backups em **Configurações → Backups** (visível apenas para administradores). Cada backup gera um arquivo `.zip` com:

- `database.sql`: dump completo do banco via **mysqldump**;
- `uploads/`: cópia dos uploads locais;
- `backup.json`: metadados do arquivo.

Os backups ficam em **BACKUP_DIR** (padrão `./data/backups`), com retenção de **BACKUP_RETENTION** arquivos. Quando **BACKUP_AUTO=true**, um backup é gerado diariamente no horário de **BACKUP_HOUR** (padrão 3h, fuso local).

Na interface é possível criar um backup manual, baixar o `.zip`, excluir arquivos antigos, importar um `.zip` gerado em outra instalação e restaurar um backup. A restauração substitui o banco e os uploads locais e exige digitar o nome do arquivo para confirmação.

Para gerar um backup manual fora da aplicação, use **mysqldump**:

```bash
mysqldump --single-transaction -u root -p documentation_center_go > backup.sql
```

Guarde também `BACKUP_DIR` e `UPLOAD_DIR`. Em instalações via PM2, esses caminhos ficam em `shared/data/` para sobreviver aos deploys.
