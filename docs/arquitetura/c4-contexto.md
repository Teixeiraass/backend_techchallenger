# C4 · Nível 1 · Contexto do sistema

Quem usa o Sistema Integrado de Atendimento e Execução de Serviços da oficina e com o que ele conversa. Notação do [C4 Model](https://c4model.com/) em Mermaid (`C4Context`).

```mermaid
C4Context
    title Oficina Mecânica · Sistema Integrado de Atendimento e Execução de Serviços

    Person(cliente, "Cliente", "Dono do veículo. Acompanha a OS e aprova ou rejeita o orçamento e os reparos adicionais pelo link, sem login.")
    Person(atendente, "Atendente", "Cadastra clientes e veículos, abre a OS, inclui itens e entrega o veículo.")
    Person(mecanico, "Mecânico", "Faz o diagnóstico, envia o orçamento, registra reparos adicionais e finaliza a OS.")
    Person(admin, "Administrador", "Gerencia catálogo, estoque e usuários; acompanha o tempo médio de execução.")

    System(oficina, "Sistema da Oficina", "API REST (Go). Clientes, veículos, catálogo de serviços e peças com estoque e o ciclo de vida da Ordem de Serviço.")

    System_Ext(email, "E-mail", "Avisa o cliente (orçamento disponível, reparo adicional, veículo pronto) e o administrador (pedido de reposição). No MVP, o envio é registrado em log.")

    Rel(cliente, oficina, "Consulta o andamento, aprova/rejeita orçamento e reparos", "HTTPS/JSON, UUID da OS + CPF/CNPJ")
    Rel(atendente, oficina, "Cadastros, abertura da OS, entrega", "HTTPS/JSON, JWT")
    Rel(mecanico, oficina, "Diagnóstico, orçamento, reparo adicional, finalização", "HTTPS/JSON, JWT")
    Rel(admin, oficina, "Catálogo, estoque, usuários, tempo médio", "HTTPS/JSON, JWT")
    Rel(oficina, email, "Envia notificações", "porta notification.Sender")
    Rel(email, cliente, "Entrega o aviso com o link de acompanhamento")
    Rel(email, admin, "Entrega o pedido de reposição")

    UpdateLayoutConfig($c4ShapeInRow="3", $c4BoundaryInRow="1")
```

## Leitura

- **Quatro pessoas, um sistema.** O cliente não tem conta: o link de acompanhamento com o UUID da OS mais o CPF/CNPJ do dono faz o papel de credencial (ADR 0005). Atendente, mecânico e administrador entram com JWT (ADR 0002) e têm papéis distintos (`RequireRole`).
- **Um sistema externo.** O e-mail é a única integração para fora. O sistema só conhece a porta `notification.Sender`; o provedor fica atrás dela (no MVP, `LogSender`). As notificações saem como reação a eventos de domínio publicados depois do commit (ADR 0006).
- **Fora do escopo do MVP:** pagamento, agenda, emissão fiscal e integração com fornecedores de peças. O "pedido de reposição" é um aviso ao administrador, não uma compra.

Próximo nível: [contêineres](c4-conteineres.md).
