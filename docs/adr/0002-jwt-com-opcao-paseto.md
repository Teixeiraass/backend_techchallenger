# ADR 0002 · JWT HS256 nas rotas administrativas, com PASETO como opção

**Status:** Aceita · **Data:** 2026-11-02

## Contexto

O enunciado pede autenticação JWT nas rotas administrativas. O grupo avaliou PASETO v2 (cifrado, sem negociação de algoritmo) como alternativa mais segura por padrão. Precisávamos também de sessão revogável (logout) e de tokens de curta duração.

## Decisão

- **JWT HS256 é o padrão** (`TOKEN_TYPE=jwt`), como pede o enunciado. O algoritmo é fixado na verificação (`WithValidMethods`): `alg=none` e outros algoritmos são recusados.
- **PASETO v2 fica como opção** (`TOKEN_TYPE=paseto`), com o mesmo `token.Maker`.
- Dois tokens: **access token** de 15 min, usado no `Authorization: Bearer`, e **refresh token** de 24 h, guardado em sessão e revogado no logout. Cada token leva o tipo (claim `typ`): o middleware só aceita `access`; `POST /tokens/refresh` só aceita `refresh`. A sessão guarda o SHA-256 do refresh token.
- Senhas com bcrypt; o login responde a mesma mensagem para e-mail inexistente e senha errada; usuário inativo perde o acesso na hora (o middleware consulta o usuário a cada requisição).
- A chave simétrica tem no mínimo 32 caracteres e, fora de `ENVIRONMENT=development`, a aplicação recusa subir com a chave de exemplo.

## Consequências

- O JWT é assinado, não cifrado: quem tiver o token lê o payload (e-mail, papel, ids). Por isso o token não leva senha nem dado do cliente e dura pouco; quem precisar de payload cifrado troca para PASETO.
- Trocar o `TOKEN_TYPE` invalida os tokens emitidos: todos fazem login de novo.
- Revogação só existe para o refresh token (sessão); um access token roubado vale até 15 min. É um risco aceito para o MVP.
