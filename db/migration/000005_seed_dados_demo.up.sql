-- Dados de demonstração para executar o fluxo da OS logo depois do "docker compose up".
-- Senhas: admin123, atendente123 e mecanico123 (hash bcrypt).
INSERT INTO "usuarios" ("nome", "email", "hashed_password", "papel") VALUES
  ('Administrador da Oficina', 'admin@oficina.com', '$2a$10$NN5q8WZcdTKPKJ7cTEnkw.SQUSH23YDWvg4EnpxXEUZRPY3GcNaJS', 'ADMINISTRADOR'),
  ('Ana Atendente', 'atendente@oficina.com', '$2a$10$0M0wQWhdcn/.vYaRIx/nluYTkgnUoXOPhDYG.qMKYELiybOMxar1q', 'ATENDENTE'),
  ('Marcos Mecânico', 'mecanico@oficina.com', '$2a$10$NwLIndRtZGVFRXFlhnLKLe9Ps96wyPq8uw/tU2CVPOeCtYiOAd/Ku', 'MECANICO');

INSERT INTO "servicos" ("nome", "descricao", "preco_centavos", "tempo_estimado_minutos") VALUES
  ('Troca de óleo', 'Troca do óleo do motor e do filtro de óleo', 8000, 30),
  ('Alinhamento', 'Alinhamento da direção', 10000, 40),
  ('Balanceamento', 'Balanceamento das quatro rodas', 8000, 40),
  ('Revisão de freios', 'Inspeção e troca de pastilhas e fluido de freio', 15000, 90),
  ('Diagnóstico eletrônico', 'Leitura de falhas com scanner', 12000, 30);

INSERT INTO "pecas" ("codigo", "nome", "descricao", "tipo", "unidade", "preco_centavos", "quantidade_estoque", "estoque_minimo") VALUES
  ('FLT-OLEO-01', 'Filtro de óleo', 'Filtro de óleo universal', 'PECA', 'UN', 3500, 20, 5),
  ('PST-FREIO-01', 'Pastilha de freio dianteira', 'Jogo de pastilhas dianteiras', 'PECA', 'JG', 12000, 8, 2),
  ('OLEO-5W30', 'Óleo 5W30 sintético', 'Óleo de motor sintético 5W30', 'INSUMO', 'L', 4500, 40, 10),
  ('FLUIDO-DOT4', 'Fluido de freio DOT4', 'Fluido de freio DOT4 500ml', 'INSUMO', 'UN', 3000, 3, 4);

INSERT INTO "clientes" ("nome", "documento", "tipo_documento", "email", "telefone") VALUES
  ('Maria Silva', '52998224725', 'CPF', 'maria.silva@email.com', '11999990000'),
  ('Transportes Exemplo LTDA', '11222333000181', 'CNPJ', 'frota@transportesexemplo.com', '1133330000');

INSERT INTO "veiculos" ("cliente_id", "placa", "marca", "modelo", "ano")
SELECT "id", 'ABC1D23', 'Volkswagen', 'Gol', 2019 FROM "clientes" WHERE "documento" = '52998224725';

INSERT INTO "veiculos" ("cliente_id", "placa", "marca", "modelo", "ano")
SELECT "id", 'XYZ1234', 'Fiat', 'Fiorino', 2021 FROM "clientes" WHERE "documento" = '11222333000181';
