package routes

type Handlers interface {
	HelperHandler
	UsuarioHandler
	ClienteHandler
	VeiculoHandler
	ServicoHandler
	PecaHandler
	OrdemServicoHandler
	AcompanhamentoHandler
}
