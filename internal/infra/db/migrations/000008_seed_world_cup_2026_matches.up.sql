-- Seed da Copa do Mundo FIFA 2026 (Canadá/México/EUA): seleções faltantes + fase de grupos.
-- Apenas os 72 jogos da fase de grupos têm seleções definidas (mata-mata depende dos
-- resultados e não pode ser referenciado por FK). Fonte do calendário: sorteio de 05/12/2025;
-- datas/estádios/sedes do cronograma divulgado em 06/12/2025. Horários convertidos de ET (EDT,
-- UTC-4) para UTC. Identificadores em inglês (ADR-0005); nomes de seleção em pt-BR (dado de domínio).

-- 1) Seleções ainda não cadastradas (as 16 do seed 000002 permanecem; Itália não está na Copa 2026).
INSERT INTO national_teams (id, name, flag_url, code) VALUES
    ('a1f3c5e7-0017-4000-8000-000000000017', 'África do Sul',                   'https://flagcdn.com/w320/za.png',     'RSA'),
    ('a1f3c5e7-0018-4000-8000-000000000018', 'Tchéquia',                        'https://flagcdn.com/w320/cz.png',     'CZE'),
    ('a1f3c5e7-0019-4000-8000-000000000019', 'Canadá',                          'https://flagcdn.com/w320/ca.png',     'CAN'),
    ('a1f3c5e7-0020-4000-8000-000000000020', 'Bósnia e Herzegovina',            'https://flagcdn.com/w320/ba.png',     'BIH'),
    ('a1f3c5e7-0021-4000-8000-000000000021', 'Catar',                           'https://flagcdn.com/w320/qa.png',     'QAT'),
    ('a1f3c5e7-0022-4000-8000-000000000022', 'Suíça',                           'https://flagcdn.com/w320/ch.png',     'SUI'),
    ('a1f3c5e7-0023-4000-8000-000000000023', 'Marrocos',                        'https://flagcdn.com/w320/ma.png',     'MAR'),
    ('a1f3c5e7-0024-4000-8000-000000000024', 'Haiti',                           'https://flagcdn.com/w320/ht.png',     'HAI'),
    ('a1f3c5e7-0025-4000-8000-000000000025', 'Escócia',                         'https://flagcdn.com/w320/gb-sct.png', 'SCO'),
    ('a1f3c5e7-0026-4000-8000-000000000026', 'Paraguai',                        'https://flagcdn.com/w320/py.png',     'PAR'),
    ('a1f3c5e7-0027-4000-8000-000000000027', 'Austrália',                       'https://flagcdn.com/w320/au.png',     'AUS'),
    ('a1f3c5e7-0028-4000-8000-000000000028', 'Turquia',                         'https://flagcdn.com/w320/tr.png',     'TUR'),
    ('a1f3c5e7-0029-4000-8000-000000000029', 'Curaçao',                         'https://flagcdn.com/w320/cw.png',     'CUW'),
    ('a1f3c5e7-0030-4000-8000-000000000030', 'Costa do Marfim',                 'https://flagcdn.com/w320/ci.png',     'CIV'),
    ('a1f3c5e7-0031-4000-8000-000000000031', 'Equador',                         'https://flagcdn.com/w320/ec.png',     'ECU'),
    ('a1f3c5e7-0032-4000-8000-000000000032', 'Suécia',                          'https://flagcdn.com/w320/se.png',     'SWE'),
    ('a1f3c5e7-0033-4000-8000-000000000033', 'Tunísia',                         'https://flagcdn.com/w320/tn.png',     'TUN'),
    ('a1f3c5e7-0034-4000-8000-000000000034', 'Egito',                           'https://flagcdn.com/w320/eg.png',     'EGY'),
    ('a1f3c5e7-0035-4000-8000-000000000035', 'Irã',                             'https://flagcdn.com/w320/ir.png',     'IRN'),
    ('a1f3c5e7-0036-4000-8000-000000000036', 'Nova Zelândia',                   'https://flagcdn.com/w320/nz.png',     'NZL'),
    ('a1f3c5e7-0037-4000-8000-000000000037', 'Cabo Verde',                      'https://flagcdn.com/w320/cv.png',     'CPV'),
    ('a1f3c5e7-0038-4000-8000-000000000038', 'Arábia Saudita',                  'https://flagcdn.com/w320/sa.png',     'KSA'),
    ('a1f3c5e7-0039-4000-8000-000000000039', 'Senegal',                         'https://flagcdn.com/w320/sn.png',     'SEN'),
    ('a1f3c5e7-0040-4000-8000-000000000040', 'Iraque',                          'https://flagcdn.com/w320/iq.png',     'IRQ'),
    ('a1f3c5e7-0041-4000-8000-000000000041', 'Noruega',                         'https://flagcdn.com/w320/no.png',     'NOR'),
    ('a1f3c5e7-0042-4000-8000-000000000042', 'Argélia',                         'https://flagcdn.com/w320/dz.png',     'ALG'),
    ('a1f3c5e7-0043-4000-8000-000000000043', 'Áustria',                         'https://flagcdn.com/w320/at.png',     'AUT'),
    ('a1f3c5e7-0044-4000-8000-000000000044', 'Jordânia',                        'https://flagcdn.com/w320/jo.png',     'JOR'),
    ('a1f3c5e7-0045-4000-8000-000000000045', 'República Democrática do Congo',  'https://flagcdn.com/w320/cd.png',     'COD'),
    ('a1f3c5e7-0046-4000-8000-000000000046', 'Uzbequistão',                     'https://flagcdn.com/w320/uz.png',     'UZB'),
    ('a1f3c5e7-0047-4000-8000-000000000047', 'Colômbia',                        'https://flagcdn.com/w320/co.png',     'COL'),
    ('a1f3c5e7-0048-4000-8000-000000000048', 'Gana',                            'https://flagcdn.com/w320/gh.png',     'GHA'),
    ('a1f3c5e7-0049-4000-8000-000000000049', 'Panamá',                          'https://flagcdn.com/w320/pa.png',     'PAN');

-- 2) Fase de grupos — 72 jogos. kickoff em UTC ('YYYY-MM-DD HH:MM:SS', formato CURRENT_TIMESTAMP).
INSERT INTO matches (id, kickoff, home_team_id, away_team_id, stadium, city, stage) VALUES
    -- 11/06
    ('wc2026-m001', '2026-06-11 19:00:00', 'a1f3c5e7-0013-4000-8000-000000000013', 'a1f3c5e7-0017-4000-8000-000000000017', 'Estadio Azteca',          'Mexico City',     'Fase de Grupos - Grupo A'),
    ('wc2026-m002', '2026-06-12 02:00:00', 'a1f3c5e7-0016-4000-8000-000000000016', 'a1f3c5e7-0018-4000-8000-000000000018', 'Estadio Akron',           'Guadalajara',     'Fase de Grupos - Grupo A'),
    -- 12/06
    ('wc2026-m003', '2026-06-12 19:00:00', 'a1f3c5e7-0019-4000-8000-000000000019', 'a1f3c5e7-0020-4000-8000-000000000020', 'BMO Field',               'Toronto',         'Fase de Grupos - Grupo B'),
    ('wc2026-m004', '2026-06-13 01:00:00', 'a1f3c5e7-0014-4000-8000-000000000014', 'a1f3c5e7-0026-4000-8000-000000000026', 'SoFi Stadium',            'Los Angeles',     'Fase de Grupos - Grupo D'),
    -- 13/06
    ('wc2026-m005', '2026-06-13 16:00:00', 'a1f3c5e7-0001-4000-8000-000000000001', 'a1f3c5e7-0023-4000-8000-000000000023', 'Gillette Stadium',        'Foxborough',      'Fase de Grupos - Grupo C'),
    ('wc2026-m006', '2026-06-13 19:00:00', 'a1f3c5e7-0024-4000-8000-000000000024', 'a1f3c5e7-0025-4000-8000-000000000025', 'MetLife Stadium',         'East Rutherford', 'Fase de Grupos - Grupo C'),
    ('wc2026-m007', '2026-06-13 22:00:00', 'a1f3c5e7-0027-4000-8000-000000000027', 'a1f3c5e7-0028-4000-8000-000000000028', 'BC Place',                'Vancouver',       'Fase de Grupos - Grupo D'),
    ('wc2026-m008', '2026-06-14 01:00:00', 'a1f3c5e7-0021-4000-8000-000000000021', 'a1f3c5e7-0022-4000-8000-000000000022', 'Levi''s Stadium',         'Santa Clara',     'Fase de Grupos - Grupo B'),
    -- 14/06
    ('wc2026-m009', '2026-06-14 16:00:00', 'a1f3c5e7-0004-4000-8000-000000000004', 'a1f3c5e7-0029-4000-8000-000000000029', 'Lincoln Financial Field', 'Philadelphia',    'Fase de Grupos - Grupo E'),
    ('wc2026-m010', '2026-06-14 19:00:00', 'a1f3c5e7-0030-4000-8000-000000000030', 'a1f3c5e7-0031-4000-8000-000000000031', 'NRG Stadium',             'Houston',         'Fase de Grupos - Grupo E'),
    ('wc2026-m011', '2026-06-14 22:00:00', 'a1f3c5e7-0008-4000-8000-000000000008', 'a1f3c5e7-0015-4000-8000-000000000015', 'AT&T Stadium',            'Arlington',       'Fase de Grupos - Grupo F'),
    ('wc2026-m012', '2026-06-15 01:00:00', 'a1f3c5e7-0032-4000-8000-000000000032', 'a1f3c5e7-0033-4000-8000-000000000033', 'Estadio BBVA',            'Monterrey',       'Fase de Grupos - Grupo F'),
    -- 15/06
    ('wc2026-m013', '2026-06-15 16:00:00', 'a1f3c5e7-0005-4000-8000-000000000005', 'a1f3c5e7-0037-4000-8000-000000000037', 'Hard Rock Stadium',       'Miami Gardens',   'Fase de Grupos - Grupo H'),
    ('wc2026-m014', '2026-06-15 19:00:00', 'a1f3c5e7-0038-4000-8000-000000000038', 'a1f3c5e7-0010-4000-8000-000000000010', 'Mercedes-Benz Stadium',   'Atlanta',         'Fase de Grupos - Grupo H'),
    ('wc2026-m015', '2026-06-15 22:00:00', 'a1f3c5e7-0011-4000-8000-000000000011', 'a1f3c5e7-0034-4000-8000-000000000034', 'SoFi Stadium',            'Los Angeles',     'Fase de Grupos - Grupo G'),
    ('wc2026-m016', '2026-06-16 01:00:00', 'a1f3c5e7-0035-4000-8000-000000000035', 'a1f3c5e7-0036-4000-8000-000000000036', 'Lumen Field',             'Seattle',         'Fase de Grupos - Grupo G'),
    -- 16/06
    ('wc2026-m017', '2026-06-16 16:00:00', 'a1f3c5e7-0003-4000-8000-000000000003', 'a1f3c5e7-0039-4000-8000-000000000039', 'MetLife Stadium',         'East Rutherford', 'Fase de Grupos - Grupo I'),
    ('wc2026-m018', '2026-06-16 19:00:00', 'a1f3c5e7-0040-4000-8000-000000000040', 'a1f3c5e7-0041-4000-8000-000000000041', 'Gillette Stadium',        'Foxborough',      'Fase de Grupos - Grupo I'),
    ('wc2026-m019', '2026-06-16 22:00:00', 'a1f3c5e7-0002-4000-8000-000000000002', 'a1f3c5e7-0042-4000-8000-000000000042', 'Arrowhead Stadium',       'Kansas City',     'Fase de Grupos - Grupo J'),
    ('wc2026-m020', '2026-06-17 01:00:00', 'a1f3c5e7-0043-4000-8000-000000000043', 'a1f3c5e7-0044-4000-8000-000000000044', 'Levi''s Stadium',         'Santa Clara',     'Fase de Grupos - Grupo J'),
    -- 17/06
    ('wc2026-m021', '2026-06-17 16:00:00', 'a1f3c5e7-0006-4000-8000-000000000006', 'a1f3c5e7-0012-4000-8000-000000000012', 'BMO Field',               'Toronto',         'Fase de Grupos - Grupo L'),
    ('wc2026-m022', '2026-06-17 19:00:00', 'a1f3c5e7-0048-4000-8000-000000000048', 'a1f3c5e7-0049-4000-8000-000000000049', 'AT&T Stadium',            'Arlington',       'Fase de Grupos - Grupo L'),
    ('wc2026-m023', '2026-06-17 22:00:00', 'a1f3c5e7-0007-4000-8000-000000000007', 'a1f3c5e7-0045-4000-8000-000000000045', 'NRG Stadium',             'Houston',         'Fase de Grupos - Grupo K'),
    ('wc2026-m024', '2026-06-18 01:00:00', 'a1f3c5e7-0046-4000-8000-000000000046', 'a1f3c5e7-0047-4000-8000-000000000047', 'Estadio Azteca',          'Mexico City',     'Fase de Grupos - Grupo K'),
    -- 18/06
    ('wc2026-m025', '2026-06-18 16:00:00', 'a1f3c5e7-0018-4000-8000-000000000018', 'a1f3c5e7-0017-4000-8000-000000000017', 'Mercedes-Benz Stadium',   'Atlanta',         'Fase de Grupos - Grupo A'),
    ('wc2026-m026', '2026-06-18 19:00:00', 'a1f3c5e7-0022-4000-8000-000000000022', 'a1f3c5e7-0020-4000-8000-000000000020', 'SoFi Stadium',            'Los Angeles',     'Fase de Grupos - Grupo B'),
    ('wc2026-m027', '2026-06-18 22:00:00', 'a1f3c5e7-0019-4000-8000-000000000019', 'a1f3c5e7-0021-4000-8000-000000000021', 'BC Place',                'Vancouver',       'Fase de Grupos - Grupo B'),
    ('wc2026-m028', '2026-06-19 01:00:00', 'a1f3c5e7-0013-4000-8000-000000000013', 'a1f3c5e7-0016-4000-8000-000000000016', 'Estadio Akron',           'Guadalajara',     'Fase de Grupos - Grupo A'),
    -- 19/06
    ('wc2026-m029', '2026-06-19 16:00:00', 'a1f3c5e7-0001-4000-8000-000000000001', 'a1f3c5e7-0024-4000-8000-000000000024', 'Lincoln Financial Field', 'Philadelphia',    'Fase de Grupos - Grupo C'),
    ('wc2026-m030', '2026-06-19 19:00:00', 'a1f3c5e7-0025-4000-8000-000000000025', 'a1f3c5e7-0023-4000-8000-000000000023', 'MetLife Stadium',         'East Rutherford', 'Fase de Grupos - Grupo C'),
    ('wc2026-m031', '2026-06-19 22:00:00', 'a1f3c5e7-0028-4000-8000-000000000028', 'a1f3c5e7-0026-4000-8000-000000000026', 'Levi''s Stadium',         'Santa Clara',     'Fase de Grupos - Grupo D'),
    ('wc2026-m032', '2026-06-20 01:00:00', 'a1f3c5e7-0014-4000-8000-000000000014', 'a1f3c5e7-0027-4000-8000-000000000027', 'Lumen Field',             'Seattle',         'Fase de Grupos - Grupo D'),
    -- 20/06
    ('wc2026-m033', '2026-06-20 16:00:00', 'a1f3c5e7-0004-4000-8000-000000000004', 'a1f3c5e7-0030-4000-8000-000000000030', 'BMO Field',               'Toronto',         'Fase de Grupos - Grupo E'),
    ('wc2026-m034', '2026-06-20 19:00:00', 'a1f3c5e7-0031-4000-8000-000000000031', 'a1f3c5e7-0029-4000-8000-000000000029', 'Arrowhead Stadium',       'Kansas City',     'Fase de Grupos - Grupo E'),
    ('wc2026-m035', '2026-06-20 22:00:00', 'a1f3c5e7-0008-4000-8000-000000000008', 'a1f3c5e7-0032-4000-8000-000000000032', 'NRG Stadium',             'Houston',         'Fase de Grupos - Grupo F'),
    ('wc2026-m036', '2026-06-21 01:00:00', 'a1f3c5e7-0033-4000-8000-000000000033', 'a1f3c5e7-0015-4000-8000-000000000015', 'Estadio BBVA',            'Monterrey',       'Fase de Grupos - Grupo F'),
    -- 21/06
    ('wc2026-m037', '2026-06-21 16:00:00', 'a1f3c5e7-0005-4000-8000-000000000005', 'a1f3c5e7-0038-4000-8000-000000000038', 'Hard Rock Stadium',       'Miami Gardens',   'Fase de Grupos - Grupo H'),
    ('wc2026-m038', '2026-06-21 19:00:00', 'a1f3c5e7-0010-4000-8000-000000000010', 'a1f3c5e7-0037-4000-8000-000000000037', 'Mercedes-Benz Stadium',   'Atlanta',         'Fase de Grupos - Grupo H'),
    ('wc2026-m039', '2026-06-21 22:00:00', 'a1f3c5e7-0011-4000-8000-000000000011', 'a1f3c5e7-0035-4000-8000-000000000035', 'SoFi Stadium',            'Los Angeles',     'Fase de Grupos - Grupo G'),
    ('wc2026-m040', '2026-06-22 01:00:00', 'a1f3c5e7-0036-4000-8000-000000000036', 'a1f3c5e7-0034-4000-8000-000000000034', 'BC Place',                'Vancouver',       'Fase de Grupos - Grupo G'),
    -- 22/06
    ('wc2026-m041', '2026-06-22 16:00:00', 'a1f3c5e7-0003-4000-8000-000000000003', 'a1f3c5e7-0040-4000-8000-000000000040', 'MetLife Stadium',         'East Rutherford', 'Fase de Grupos - Grupo I'),
    ('wc2026-m042', '2026-06-22 19:00:00', 'a1f3c5e7-0041-4000-8000-000000000041', 'a1f3c5e7-0039-4000-8000-000000000039', 'Lincoln Financial Field', 'Philadelphia',    'Fase de Grupos - Grupo I'),
    ('wc2026-m043', '2026-06-22 22:00:00', 'a1f3c5e7-0002-4000-8000-000000000002', 'a1f3c5e7-0043-4000-8000-000000000043', 'AT&T Stadium',            'Arlington',       'Fase de Grupos - Grupo J'),
    ('wc2026-m044', '2026-06-23 01:00:00', 'a1f3c5e7-0044-4000-8000-000000000044', 'a1f3c5e7-0042-4000-8000-000000000042', 'Levi''s Stadium',         'Santa Clara',     'Fase de Grupos - Grupo J'),
    -- 23/06
    ('wc2026-m045', '2026-06-23 16:00:00', 'a1f3c5e7-0006-4000-8000-000000000006', 'a1f3c5e7-0048-4000-8000-000000000048', 'Gillette Stadium',        'Foxborough',      'Fase de Grupos - Grupo L'),
    ('wc2026-m046', '2026-06-23 19:00:00', 'a1f3c5e7-0049-4000-8000-000000000049', 'a1f3c5e7-0012-4000-8000-000000000012', 'BMO Field',               'Toronto',         'Fase de Grupos - Grupo L'),
    ('wc2026-m047', '2026-06-23 22:00:00', 'a1f3c5e7-0007-4000-8000-000000000007', 'a1f3c5e7-0046-4000-8000-000000000046', 'NRG Stadium',             'Houston',         'Fase de Grupos - Grupo K'),
    ('wc2026-m048', '2026-06-24 01:00:00', 'a1f3c5e7-0047-4000-8000-000000000047', 'a1f3c5e7-0045-4000-8000-000000000045', 'Estadio Akron',           'Guadalajara',     'Fase de Grupos - Grupo K'),
    -- 24/06 (rodada final dos Grupos A, B e C — jogos simultâneos)
    ('wc2026-m049', '2026-06-24 16:00:00', 'a1f3c5e7-0013-4000-8000-000000000013', 'a1f3c5e7-0018-4000-8000-000000000018', 'Estadio Azteca',          'Mexico City',     'Fase de Grupos - Grupo A'),
    ('wc2026-m050', '2026-06-24 19:00:00', 'a1f3c5e7-0016-4000-8000-000000000016', 'a1f3c5e7-0017-4000-8000-000000000017', 'Estadio BBVA',            'Monterrey',       'Fase de Grupos - Grupo A'),
    ('wc2026-m051', '2026-06-24 22:00:00', 'a1f3c5e7-0019-4000-8000-000000000019', 'a1f3c5e7-0022-4000-8000-000000000022', 'BC Place',                'Vancouver',       'Fase de Grupos - Grupo B'),
    ('wc2026-m052', '2026-06-25 01:00:00', 'a1f3c5e7-0020-4000-8000-000000000020', 'a1f3c5e7-0021-4000-8000-000000000021', 'Lumen Field',             'Seattle',         'Fase de Grupos - Grupo B'),
    ('wc2026-m053', '2026-06-24 16:00:00', 'a1f3c5e7-0025-4000-8000-000000000025', 'a1f3c5e7-0001-4000-8000-000000000001', 'Hard Rock Stadium',       'Miami Gardens',   'Fase de Grupos - Grupo C'),
    ('wc2026-m054', '2026-06-24 19:00:00', 'a1f3c5e7-0023-4000-8000-000000000023', 'a1f3c5e7-0024-4000-8000-000000000024', 'Mercedes-Benz Stadium',   'Atlanta',         'Fase de Grupos - Grupo C'),
    -- 25/06 (rodada final dos Grupos E, F e D)
    ('wc2026-m055', '2026-06-25 16:00:00', 'a1f3c5e7-0031-4000-8000-000000000031', 'a1f3c5e7-0004-4000-8000-000000000004', 'Lincoln Financial Field', 'Philadelphia',    'Fase de Grupos - Grupo E'),
    ('wc2026-m056', '2026-06-25 19:00:00', 'a1f3c5e7-0029-4000-8000-000000000029', 'a1f3c5e7-0030-4000-8000-000000000030', 'MetLife Stadium',         'East Rutherford', 'Fase de Grupos - Grupo E'),
    ('wc2026-m057', '2026-06-25 22:00:00', 'a1f3c5e7-0033-4000-8000-000000000033', 'a1f3c5e7-0008-4000-8000-000000000008', 'AT&T Stadium',            'Arlington',       'Fase de Grupos - Grupo F'),
    ('wc2026-m058', '2026-06-26 01:00:00', 'a1f3c5e7-0015-4000-8000-000000000015', 'a1f3c5e7-0032-4000-8000-000000000032', 'Arrowhead Stadium',       'Kansas City',     'Fase de Grupos - Grupo F'),
    ('wc2026-m059', '2026-06-26 02:00:00', 'a1f3c5e7-0014-4000-8000-000000000014', 'a1f3c5e7-0028-4000-8000-000000000028', 'SoFi Stadium',            'Los Angeles',     'Fase de Grupos - Grupo D'),
    ('wc2026-m060', '2026-06-25 19:00:00', 'a1f3c5e7-0026-4000-8000-000000000026', 'a1f3c5e7-0027-4000-8000-000000000027', 'Levi''s Stadium',         'Santa Clara',     'Fase de Grupos - Grupo D'),
    -- 26/06 (rodada final dos Grupos I, G e H)
    ('wc2026-m061', '2026-06-26 16:00:00', 'a1f3c5e7-0041-4000-8000-000000000041', 'a1f3c5e7-0003-4000-8000-000000000003', 'Gillette Stadium',        'Foxborough',      'Fase de Grupos - Grupo I'),
    ('wc2026-m062', '2026-06-26 19:00:00', 'a1f3c5e7-0039-4000-8000-000000000039', 'a1f3c5e7-0040-4000-8000-000000000040', 'BMO Field',               'Toronto',         'Fase de Grupos - Grupo I'),
    ('wc2026-m063', '2026-06-26 22:00:00', 'a1f3c5e7-0036-4000-8000-000000000036', 'a1f3c5e7-0011-4000-8000-000000000011', 'Lumen Field',             'Seattle',         'Fase de Grupos - Grupo G'),
    ('wc2026-m064', '2026-06-27 01:00:00', 'a1f3c5e7-0034-4000-8000-000000000034', 'a1f3c5e7-0035-4000-8000-000000000035', 'BC Place',                'Vancouver',       'Fase de Grupos - Grupo G'),
    ('wc2026-m065', '2026-06-26 16:00:00', 'a1f3c5e7-0010-4000-8000-000000000010', 'a1f3c5e7-0005-4000-8000-000000000005', 'NRG Stadium',             'Houston',         'Fase de Grupos - Grupo H'),
    ('wc2026-m066', '2026-06-26 19:00:00', 'a1f3c5e7-0037-4000-8000-000000000037', 'a1f3c5e7-0038-4000-8000-000000000038', 'Estadio Akron',           'Guadalajara',     'Fase de Grupos - Grupo H'),
    -- 27/06 (rodada final dos Grupos L, J e K)
    ('wc2026-m067', '2026-06-27 16:00:00', 'a1f3c5e7-0049-4000-8000-000000000049', 'a1f3c5e7-0006-4000-8000-000000000006', 'MetLife Stadium',         'East Rutherford', 'Fase de Grupos - Grupo L'),
    ('wc2026-m068', '2026-06-27 19:00:00', 'a1f3c5e7-0012-4000-8000-000000000012', 'a1f3c5e7-0048-4000-8000-000000000048', 'Lincoln Financial Field', 'Philadelphia',    'Fase de Grupos - Grupo L'),
    ('wc2026-m069', '2026-06-27 22:00:00', 'a1f3c5e7-0044-4000-8000-000000000044', 'a1f3c5e7-0002-4000-8000-000000000002', 'Arrowhead Stadium',       'Kansas City',     'Fase de Grupos - Grupo J'),
    ('wc2026-m070', '2026-06-28 01:00:00', 'a1f3c5e7-0042-4000-8000-000000000042', 'a1f3c5e7-0043-4000-8000-000000000043', 'AT&T Stadium',            'Arlington',       'Fase de Grupos - Grupo J'),
    ('wc2026-m071', '2026-06-27 16:00:00', 'a1f3c5e7-0047-4000-8000-000000000047', 'a1f3c5e7-0007-4000-8000-000000000007', 'Hard Rock Stadium',       'Miami Gardens',   'Fase de Grupos - Grupo K'),
    ('wc2026-m072', '2026-06-27 19:00:00', 'a1f3c5e7-0045-4000-8000-000000000045', 'a1f3c5e7-0046-4000-8000-000000000046', 'Mercedes-Benz Stadium',   'Atlanta',         'Fase de Grupos - Grupo K');
