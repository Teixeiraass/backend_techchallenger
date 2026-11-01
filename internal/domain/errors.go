// Package domain reúne as regras de negócio da oficina que não dependem de
// banco, HTTP ou frameworks: objetos de valor, invariantes dos agregados e a
// máquina de estados da Ordem de Serviço. Os serviços da camada de aplicação
// (internal/service) carregam os dados, aplicam estas regras e persistem.
package domain

import "errors"

// Kind classifica o erro de domínio para a camada HTTP escolher o status.
type Kind int

const (
	// KindValidation indica dado de entrada inválido (HTTP 400).
	KindValidation Kind = iota + 1
	// KindConflict indica regra de negócio violada pelo estado atual (HTTP 409).
	KindConflict
	// KindNotFound indica recurso inexistente (HTTP 404).
	KindNotFound
	// KindForbidden indica ação não permitida para quem a solicitou (HTTP 403).
	KindForbidden
	// KindUnauthorized indica credenciais ou sessão inválidas (HTTP 401).
	KindUnauthorized
)

// Error é o erro de negócio devolvido pelo domínio e pelos serviços.
type Error struct {
	Kind    Kind
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// Is permite comparar erros pelo código com errors.Is.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

func NewValidationError(code, message string) *Error {
	return &Error{Kind: KindValidation, Code: code, Message: message}
}

func NewConflictError(code, message string) *Error {
	return &Error{Kind: KindConflict, Code: code, Message: message}
}

func NewNotFoundError(code, message string) *Error {
	return &Error{Kind: KindNotFound, Code: code, Message: message}
}

func NewForbiddenError(code, message string) *Error {
	return &Error{Kind: KindForbidden, Code: code, Message: message}
}

func NewUnauthorizedError(code, message string) *Error {
	return &Error{Kind: KindUnauthorized, Code: code, Message: message}
}

// KindOf devolve o Kind do erro, ou 0 quando não é um erro de domínio.
func KindOf(err error) Kind {
	var domainErr *Error
	if errors.As(err, &domainErr) {
		return domainErr.Kind
	}
	return 0
}
