## Resumo

<!-- O que muda e por quê, em duas ou três frases. -->

## Crítica ou issue relacionada

<!-- Ex.: crítica 5.1 da avaliação, ou #12. -->

## Como testar

```bash
docker compose up -d postgres
make test
```

<!-- Passos extras, rotas para chamar, dados de exemplo. -->

## Definition of Done

- [ ] Testes passam com banco (`make test`).
- [ ] Cobertura dos domínios críticos ≥ 80% (`make coverage`).
- [ ] `make swag` rodado se a API mudou (handlers ou DTOs).
- [ ] Linguagem ubíqua / README atualizados se uma regra de negócio mudou.
- [ ] Análise de segurança sem achado novo (`make vulncheck`, `make gosec`, `make trivy`).
- [ ] Um revisor aprovou.
