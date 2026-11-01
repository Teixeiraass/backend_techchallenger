package dto

// PaginacaoQuery is embedded in list queries (?pagina=1&tamanho=20).
type PaginacaoQuery struct {
	Pagina  int32 `form:"pagina,default=1" binding:"min=1"`
	Tamanho int32 `form:"tamanho,default=20" binding:"min=1,max=100"`
}

func (p PaginacaoQuery) Limite() int32 {
	if p.Tamanho <= 0 {
		return 20
	}
	return p.Tamanho
}

func (p PaginacaoQuery) Deslocamento() int32 {
	if p.Pagina <= 1 {
		return 0
	}
	return (p.Pagina - 1) * p.Limite()
}
