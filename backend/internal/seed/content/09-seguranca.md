# Segurança

- Senhas são armazenadas com **bcrypt**.
- Sessões e tokens são salvos apenas como hash SHA-256.
- Cookies de sessão são HttpOnly, SameSite=Lax e marcados Secure em produção.
- Requisições mutáveis com sessão exigem o header **X-CSRF-Token**.
- Login e cadastro têm limite de 5 tentativas por minuto por e-mail/IP.
- Markdown é renderizado com HTML bruto desabilitado e sanitização adicional.
- Uploads validam tipo real do arquivo (magic bytes) e extensão, com limite configurável.
- Credenciais de armazenamento são criptografadas com AES-GCM usando APP_KEY.
- Rotas administrativas validam **is_admin** no servidor, não apenas na interface.
