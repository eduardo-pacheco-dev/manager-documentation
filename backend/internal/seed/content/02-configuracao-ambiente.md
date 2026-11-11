# Configuração do ambiente

## Requisitos

- Go 1.26 ou superior.
- Node.js 20.19+ e npm.
- MySQL 8.0+ (ou MariaDB compatível) em execução. O banco é criado automaticamente se o usuário tiver permissão.

## Backend

```bash
cd backend
cp .env.example .env
go run ./cmd/server
```

Na primeira execução o banco é migrado e um administrador é criado com as credenciais definidas em **SEED_ADMIN_EMAIL** e **SEED_ADMIN_PASSWORD**. O padrão local é **admin@example.com / password**.

## Frontend

```bash
cd frontend
npm install
npm run dev
```

O Vite roda em **http://localhost:5173** e encaminha chamadas de /api e /uploads para o backend em **:8080**.

## Produção

```bash
cd frontend && npm run build
cd ../backend && STATIC_DIR=../frontend/dist go run ./cmd/server
```
