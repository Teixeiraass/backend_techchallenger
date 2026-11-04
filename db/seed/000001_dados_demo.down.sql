DELETE FROM "veiculos" WHERE "placa" IN ('ABC1D23', 'XYZ1234');
DELETE FROM "clientes" WHERE "documento" IN ('52998224725', '11222333000181');
DELETE FROM "pecas" WHERE "codigo" IN ('FLT-OLEO-01', 'PST-FREIO-01', 'OLEO-5W30', 'FLUIDO-DOT4');
DELETE FROM "servicos" WHERE "nome" IN ('Troca de óleo', 'Alinhamento', 'Balanceamento', 'Revisão de freios', 'Diagnóstico eletrônico');
DELETE FROM "usuarios" WHERE "email" IN ('admin@oficina.com', 'atendente@oficina.com', 'mecanico@oficina.com');
