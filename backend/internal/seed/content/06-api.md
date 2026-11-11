# API de autenticação

Prefixo: **/api/v1**.

| Método | Rota | Descrição |
| --- | --- | --- |
| POST | /auth/signup | Cria conta e retorna token quando device_name é enviado. |
| POST | /auth/login | Autentica e retorna token. |
| POST | /auth/logout | Revoga o token atual. |
| GET | /auth/me | Retorna o usuário autenticado. |
| POST | /uploads | Envia uma imagem (multipart file). |

## Exemplo

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password","device_name":"cli"}'
```

A resposta inclui **data.user** e **data.token**. Use o token em **Authorization: Bearer ...**.

## Uploads

O endpoint aceita imagens JPG, PNG, GIF, WEBP e BMP. O tipo real é validado pelos magic bytes. O limite padrão é 20 MB e pode ser alterado em Configurações > Armazenamento.
