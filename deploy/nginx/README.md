# Nginx + HTTPS para afldoc.com.br

Arquivos:

- `afldoc.com.br.http.conf` — configuração temporária HTTP para validar o proxy e emitir o certificado.
- `afldoc.com.br.conf` — configuração final com HTTPS, HSTS e redirecionamento de `www`.

Pressupostos:

- Aplicação rodando em `127.0.0.1:8080` via PM2.
- DNS `afldoc.com.br` e `www.afldoc.com.br` apontando para o IP da VPS.
- Acesso como `root` (ajuste com `sudo` se necessário).

## 1. DNS

No provedor do domínio, configure:

| Tipo | Nome | Valor |
| --- | --- | --- |
| A | `@` | IP da VPS |
| A | `www` | IP da VPS |

Confirme antes de continuar:

```bash
dig +short afldoc.com.br
dig +short www.afldoc.com.br
```

## 2. Instalar Nginx e Certbot

```bash
apt-get update
apt-get install -y nginx certbot
mkdir -p /var/www/certbot
```

## 3. Subir a configuração HTTP temporária

```bash
cd ~/app/manager-documentation

install -m 644 deploy/nginx/afldoc.com.br.http.conf /etc/nginx/sites-available/afldoc.com.br
ln -sf /etc/nginx/sites-available/afldoc.com.br /etc/nginx/sites-enabled/afldoc.com.br
rm -f /etc/nginx/sites-enabled/default

nginx -t
systemctl enable --now nginx
systemctl reload nginx
```

Teste `http://afldoc.com.br` antes de emitir o certificado.

## 4. Emitir o certificado Let's Encrypt

```bash
certbot certonly --webroot -w /var/www/certbot \
  --cert-name afldoc.com.br \
  -d afldoc.com.br -d www.afldoc.com.br \
  --email admin@afldoc.com.br \
  --agree-tos --no-eff-email
```

O certificado será criado em `/etc/letsencrypt/live/afldoc.com.br/`.

## 5. Ativar a configuração HTTPS final

```bash
cd ~/app/manager-documentation

install -m 644 deploy/nginx/afldoc.com.br.conf /etc/nginx/sites-available/afldoc.com.br
ln -sf /etc/nginx/sites-available/afldoc.com.br /etc/nginx/sites-enabled/afldoc.com.br

nginx -t
systemctl reload nginx
```

Acesse `https://afldoc.com.br`. `www.afldoc.com.br` redireciona para o domínio principal.

## 6. Renovação automática

O Certbot já instala um timer de renovação. Garanta que o Nginx recarregue após renovar:

```bash
install -d /etc/letsencrypt/renewal-hooks/deploy
cat > /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh <<'EOF'
#!/bin/sh
systemctl reload nginx
EOF
chmod +x /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh

certbot renew --dry-run
```

## 7. Configuração da aplicação

No `.env` da aplicação (`/root/app/manager-documentation/shared/.env`, ou o diretório configurado):

```env
PUBLIC_URL=https://afldoc.com.br
ALLOWED_ORIGINS=https://afldoc.com.br,https://www.afldoc.com.br
SECURE_COOKIES=true
```

Reinicie o PM2:

```bash
pm2 restart documentation-center --update-env
```

Se estiver subindo o setup do zero, use direto:

```bash
bash deploy.sh setup --url https://afldoc.com.br --admin-email admin@afldoc.com.br
```

Recomendado: faça o app escutar apenas em localhost e deixe o Nginx como única entrada pública. No GitHub Actions, configure `APP_BIND=127.0.0.1`; ou edite o ecosystem/`.env` e reinicie.

## 8. Firewall

```bash
ufw allow 'Nginx Full'
# se a porta 8080 estiver liberada, remova:
# ufw delete allow 8080/tcp
```

## Observações

- `client_max_body_size 50m` cobre os anexos Nokia de até 40 MB.
- Os timeouts de proxy são de 300s para uploads grandes.
- HSTS está habilitado por um ano. Só mantenha se o HTTPS estiver estável.
- Logs: `/var/log/nginx/afldoc.access.log` e `/var/log/nginx/afldoc.error.log`.
