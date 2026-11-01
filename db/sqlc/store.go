package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Store provides all functions to execute db queries and transactions
type Store interface {
	Querier
	// ExecTx runs fn inside a database transaction. Every query issued through
	// the Querier received by fn is committed together, or rolled back if fn
	// returns an error.
	ExecTx(ctx context.Context, fn func(Querier) error) error
}

// SqlStore provides all functions to execute SQL queries and transaction
type SqlStore struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &SqlStore{
		db:      db,
		Queries: New(db),
	}
}

func (store *SqlStore) ExecTx(ctx context.Context, fn func(Querier) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(New(tx)); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}
