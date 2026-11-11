# Banco de dados e migrações

O banco padrão é SQLite, configurado por **DATABASE_PATH** (padrão **./data/app.db**).

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

Com o servidor parado, copie **data/app.db**. Durante a execução, use a API de backup do SQLite ou faça uma cópia consistente com **VACUUM INTO**.
