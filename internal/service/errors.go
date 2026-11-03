package service

import (
	"database/sql"
	"errors"

	"github.com/Teixeiraass/backend_techchallenger/internal/domain"
	"github.com/lib/pq"
)

var (
	ErrUsuarioNaoEncontrado   = domain.NewNotFoundError("USUARIO_NAO_ENCONTRADO", "usuário não encontrado")
	ErrEmailDuplicado         = domain.NewConflictError("EMAIL_DUPLICADO", "já existe um usuário com este e-mail")
	ErrCredenciaisInvalidas   = domain.NewUnauthorizedError("CREDENCIAIS_INVALIDAS", "e-mail ou senha inválidos")
	ErrUsuarioInativo         = domain.NewForbiddenError("USUARIO_INATIVO", "usuário inativo")
	ErrSessaoInvalida         = domain.NewUnauthorizedError("SESSAO_INVALIDA", "sessão inválida ou expirada")
	ErrClienteNaoEncontrado   = domain.NewNotFoundError("CLIENTE_NAO_ENCONTRADO", "cliente não encontrado")
	ErrDocumentoDuplicado     = domain.NewConflictError("DOCUMENTO_DUPLICADO", "já existe um cliente com este CPF/CNPJ")
	ErrClienteEmUso           = domain.NewConflictError("CLIENTE_EM_USO", "o cliente possui veículos ou ordens de serviço e não pode ser removido")
	ErrVeiculoNaoEncontrado   = domain.NewNotFoundError("VEICULO_NAO_ENCONTRADO", "veículo não encontrado")
	ErrPlacaDuplicada         = domain.NewConflictError("PLACA_DUPLICADA", "já existe um veículo com esta placa")
	ErrVeiculoEmUso           = domain.NewConflictError("VEICULO_EM_USO", "o veículo possui ordens de serviço e não pode ser removido")
	ErrVeiculoDeOutroCliente  = domain.NewConflictError("VEICULO_DE_OUTRO_CLIENTE", "o veículo informado não pertence ao cliente")
	ErrServicoNaoEncontrado   = domain.NewNotFoundError("SERVICO_NAO_ENCONTRADO", "serviço não encontrado")
	ErrServicoDuplicado       = domain.NewConflictError("SERVICO_DUPLICADO", "já existe um serviço com este nome")
	ErrPecaNaoEncontrada      = domain.NewNotFoundError("PECA_NAO_ENCONTRADA", "peça ou insumo não encontrado")
	ErrCodigoPecaDuplicado    = domain.NewConflictError("CODIGO_PECA_DUPLICADO", "já existe uma peça com este código")
	ErrOSNaoEncontrada        = domain.NewNotFoundError("OS_NAO_ENCONTRADA", "ordem de serviço não encontrada")
	ErrDataInvalida           = domain.NewValidationError("DATA_INVALIDA", "data inválida: use o formato AAAA-MM-DD")
	ErrIntervaloDatasInvalido = domain.NewValidationError("INTERVALO_INVALIDO", "data_inicio deve ser anterior a data_fim")
)

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation"
}

func isForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation"
}

// notFound troca sql.ErrNoRows pelo erro de domínio do recurso.
func notFound(err error, target error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return target
	}
	return err
}
