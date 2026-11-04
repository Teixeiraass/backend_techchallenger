# Análise de vulnerabilidades

Data da análise: 03/11/2026 (relatórios regerados após as correções de autenticação, limite de tentativas e log de acesso). Os relatórios brutos estão nesta pasta e são gerados de novo pelos comandos indicados. O CI (`.github/workflows/ci.yml`, job `security`) roda as mesmas ferramentas a cada push e PR.

## Resumo

| Ferramenta | O que analisa | Comando | Resultado |
|---|---|---|---|
| [govulncheck](https://go.dev/blog/govulncheck) | Dependências Go e biblioteca padrão, considerando o que o código realmente chama | `make vulncheck` | **0 vulnerabilidades alcançáveis** ([govulncheck.txt](govulncheck.txt)) |
| [gosec](https://github.com/securego/gosec) | Análise estática do código (SAST) | `make gosec` | **0 problemas** ([gosec.txt](gosec.txt)) |
| [Trivy](https://trivy.dev) | Imagem Docker: pacotes do Alpine e binários Go | `make trivy` | Alpine: **0**. Binário `migrate`: **0**. Binário da API: **1 aviso sem correção, não alcançável** ([trivy-image.txt](trivy-image.txt)) |

## O que foi encontrado e corrigido

1. **Dependências com vulnerabilidades alcançáveis** (govulncheck): `golang.org/x/net` v0.55.0 (GO-2026-6603, 6611, 6612, 6617, HTTP/2) e `github.com/quic-go/quic-go` v0.59.0 (GO-2026-5676), ambas trazidas pelo Gin. Atualizadas para `x/net` v0.60.0 e `quic-go` v0.59.1.
2. **`golang.org/x/net` v0.60.0** (govulncheck e Trivy, 03/11): GO-2026-6866 e GO-2026-6860 no parser de HTML, não alcançáveis pelo código. Atualizado para v0.62.0.
3. **Biblioteca padrão do Go**: a versão 1.26.0 tinha 32 vulnerabilidades corrigidas em patches posteriores. O `go.mod` fixa `toolchain go1.26.10` e o Dockerfile usa `golang:1.26.10-alpine3.23`.
4. **Binário `migrate` da imagem** (Trivy): o binário pré-compilado do golang-migrate trazia 114 vulnerabilidades (4 críticas e 51 altas), vindas de Go e drivers antigos. Agora o Dockerfile compila o `migrate` v4.20.1 com o mesmo Go 1.26.10 e só com o driver postgres. Resultado: 0.
5. **gosec**:
   - G104: erros do `viper.BindEnv` eram ignorados. Agora são tratados.
   - G404/G115: o gerador de dados de teste (`util/random.go`) usa `math/rand`. Esse código não é usado em segurança e está marcado com `#nosec` e justificativa. A conversão `int -> byte` foi trocada por `strconv`.

## Risco aceito

| Achado | Motivo |
|---|---|
| `golang.org/x/crypto` GO-2026-5932: o pacote `openpgp` está descontinuado | Não existe versão publicada com a correção (só um commit sem tag, `v0.58.1-0.20261009181718-2f1f834330b2`). A API não importa `openpgp`, e o govulncheck confirma que o código não chama o pacote afetado. Usamos só `bcrypt` e `chacha20poly1305` do mesmo módulo. |

## Controles de segurança da aplicação

- **Autenticação**: JWT HS256 por padrão (`TOKEN_TYPE=jwt`), com opção de PASETO v2 (`TOKEN_TYPE=paseto`). O algoritmo é fixado na verificação, então `alg=none` e outros algoritmos são recusados. O access token expira em 15 min. Todo token leva o tipo (claim `typ`): o middleware só aceita `access` (refresh token em rota protegida responde 401 `TOKEN_TIPO_INVALIDO`) e `POST /tokens/refresh` só aceita `refresh`. O refresh token fica salvo em sessão como SHA-256 (nunca em texto puro) e é revogado no logout: depois dele nem renova nem autentica. Há teste de integração para cada caso (`internal/integration/seguranca_test.go`). O JWT é assinado, mas não é cifrado: quem tiver o token consegue ler o payload (e-mail, papel e IDs do usuário). Por isso o token não leva senha nem dado do cliente e dura pouco. Quem precisar do payload cifrado usa `TOKEN_TYPE=paseto`. Trocar o `TOKEN_TYPE` invalida os tokens já emitidos: os usuários precisam fazer login de novo.
- **Autorização por papel**: ADMINISTRADOR, ATENDENTE e MECANICO, aplicada por middleware em cada rota administrativa. As respostas do middleware são em português e trazem um `code` (`TOKEN_AUSENTE`, `TOKEN_FORMATO_INVALIDO`, `TOKEN_INVALIDO`, `TOKEN_EXPIRADO`, `TOKEN_TIPO_INVALIDO`, `USUARIO_NAO_ENCONTRADO`, `USUARIO_INATIVO`, `PAPEL_SEM_PERMISSAO`).
- **Limite de tentativas**: login, renovação de token e as rotas públicas de acompanhamento têm limite por IP (`RATE_LIMIT_RPS`, `RATE_LIMIT_BURST`, padrão 1 req/s e rajada 10). Acima disso: 429 `MUITAS_TENTATIVAS` com `Retry-After`. Senha de usuário tem no mínimo 8 caracteres.
- **Dado pessoal fora do log**: o log de acesso registra método, caminho sem query string, status e latência. O CPF/CNPJ vai no header `X-Documento` (consulta pública e `GET /clientes/por-documento`), não na URL. O parâmetro `?documento=` da consulta pública continua aceito por uma versão (obsoleto) e, mesmo assim, não aparece no log.
- **Servidor HTTP**: `ReadHeaderTimeout` 5 s, `ReadTimeout` 15 s, `WriteTimeout` 30 s, `IdleTimeout` 60 s (contra Slowloris), corpo limitado a 1 MiB (413 `CORPO_GRANDE_DEMAIS`) e encerramento gracioso em `SIGTERM`.
- **Seed de demonstração fora das migrations**: os usuários `admin@oficina.com` etc. ficam em `db/seed` e só entram com `SEED_DEMO=true` (ligado só no `docker-compose.yaml` local). Uma imagem publicada em outro ambiente não ganha administrador com senha conhecida.
- **Segredo de exemplo recusado**: fora de `ENVIRONMENT=development` a aplicação não sobe com a chave de exemplo nem com chave menor que 32 caracteres.
- **Usuário desativado** perde o acesso na hora: o middleware consulta o usuário a cada requisição.
- **Senhas** guardadas com bcrypt. O login responde com a mesma mensagem para e-mail inexistente e para senha errada.
- **Rotas públicas do cliente** exigem o UUID da OS e o CPF/CNPJ do dono. Documento errado recebe 404, igual a OS inexistente, para não revelar quais OS existem.
- **Validação de entrada**: CPF/CNPJ com dígitos verificadores, placa nos padrões antigo e Mercosul, e binding com limites de tamanho. Erros de validação voltam com o nome JSON do campo e a regra (`campos: [{campo, regra}]`), sem expor nomes de struct.
- **Erros 500** voltam com mensagem genérica. O detalhe vai só para o log.
- **SQL** só pelo sqlc, com queries parametrizadas.
- **Container** roda como usuário sem privilégios (`app`).
- **Segredos** ficam fora do repositório: `app.env` está no `.gitignore`. O `app.env.example` e o `docker-compose.yaml` têm só valores de desenvolvimento local, identificados como tal.
