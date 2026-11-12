# Deploy

## Build

```bash
cd frontend && npm ci && npm run build
cd ../backend && CGO_ENABLED=0 go build -o documentation-center ./cmd/server
```

## Execução

```bash
STATIC_DIR=../frontend/dist \
DB_HOST=127.0.0.1 \
DB_PORT=3306 \
DB_NAME=documentation_center_go \
DB_USER=documentation \
DB_PASSWORD=troque-esta-senha \
UPLOAD_DIR=/var/lib/documentation/uploads \
PUBLIC_URL=https://docs.example.com \
SECURE_COOKIES=true \
./documentation-center
```

## Administrador

O setup cria o administrador inicial. Para criar outro administrador ou promover uma conta existente na VPS:

```bash
cd ~/app/manager-documentation
sudo bash deploy.sh admin --email admin2@exemplo.com --password "senha-forte" --name "Admin"
```

Sem `--password`, uma senha aleatória é gerada e impressa. O e-mail existente é promovido e ativado; a senha só muda se `--password` for informado.

## Docker

```bash
docker compose up --build
```

## Variáveis importantes

| Variável | Descrição |
| --- | --- |
| ADDR | Endereço HTTP (padrão :8080). |
| DB_HOST | Host do MySQL (padrão 127.0.0.1). |
| DB_PORT | Porta do MySQL (padrão 3306). |
| DB_NAME | Nome do banco (padrão documentation_center_go). |
| DB_USER | Usuário do MySQL. |
| DB_PASSWORD | Senha do MySQL. |
| DB_DSN | DSN completa opcional, sobrepõe as variáveis DB_*. |
| UPLOAD_DIR | Diretório de uploads locais. |
| STATIC_DIR | Build do frontend servido pelo Go. |
| PUBLIC_URL | URL pública usada em links de upload. |
| SECURE_COOKIES | Cookies apenas via HTTPS. |
| APP_KEY | Chave usada para criptografar credenciais de armazenamento. |
