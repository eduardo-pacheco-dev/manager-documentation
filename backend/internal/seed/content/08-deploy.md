# Deploy

## Build

```bash
cd frontend && npm ci && npm run build
cd ../backend && CGO_ENABLED=0 go build -o documentation-center ./cmd/server
```

## Execução

```bash
STATIC_DIR=../frontend/dist \
DATABASE_PATH=/var/lib/documentation/app.db \
UPLOAD_DIR=/var/lib/documentation/uploads \
PUBLIC_URL=https://docs.example.com \
SECURE_COOKIES=true \
./documentation-center
```

## Docker

```bash
docker compose up --build
```

## Variáveis importantes

| Variável | Descrição |
| --- | --- |
| ADDR | Endereço HTTP (padrão :8080). |
| DATABASE_PATH | Caminho do banco SQLite. |
| UPLOAD_DIR | Diretório de uploads locais. |
| STATIC_DIR | Build do frontend servido pelo Go. |
| PUBLIC_URL | URL pública usada em links de upload. |
| SECURE_COOKIES | Cookies apenas via HTTPS. |
| APP_KEY | Chave usada para criptografar credenciais de armazenamento. |
