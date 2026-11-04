// Package repository implementa as portas de persistência do domínio
// (ordemservico.Repositorio, estoque.Repositorio) e da aplicação
// (service.UnidadeDeTrabalho, service.Catalogo) com o sqlc sobre o
// PostgreSQL. É a única camada que conhece as tabelas.
package repository

import (
	"context"
	"database/sql"
	"errors"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/estoque"
	"github.com/Teixeiraass/backend_techchallenger/internal/domain/ordemservico"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
)

// UnidadeDeTrabalho abre a transação pelo Store e entrega os repositórios
// ligados ao Querier da transação.
type UnidadeDeTrabalho struct {
	store db.Store
}

func NewUnidadeDeTrabalho(store db.Store) *UnidadeDeTrabalho {
	return &UnidadeDeTrabalho{store: store}
}

func (u *UnidadeDeTrabalho) Executar(ctx context.Context, fn func(service.Repositorios) error) error {
	return u.store.ExecTx(ctx, func(q db.Querier) error {
		return fn(repositorios{q: q})
	})
}

func (u *UnidadeDeTrabalho) Leitura() service.Repositorios {
	return repositorios{q: u.store}
}

type repositorios struct {
	q db.Querier
}

func (r repositorios) OrdemServico() ordemservico.Repositorio {
	return &OrdemServicoRepositorio{q: r.q}
}
func (r repositorios) Pecas() estoque.Repositorio { return &PecaRepositorio{q: r.q} }
func (r repositorios) Catalogo() service.Catalogo { return &Catalogo{q: r.q} }

// naoEncontrado troca sql.ErrNoRows pelo erro de domínio do recurso.
func naoEncontrado(err error, alvo error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return alvo
	}
	return err
}
