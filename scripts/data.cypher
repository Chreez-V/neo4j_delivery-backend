// Limpieza inicial (opcional)
MATCH (n) DETACH DELETE n;

// Creación de nodos
CREATE (cd1:CentroDistribucion:Zona {nombre: 'Centro Principal', tipo_zona: 'logistica', capacidad_vehiculos: 50});
CREATE (cd2:CentroDistribucion:Zona {nombre: 'Centro Secundario', tipo_zona: 'logistica', capacidad_vehiculos: 30});

CREATE (z1:Zona {nombre: 'AltaVista', tipo_zona: 'comercial'});
CREATE (z2:Zona {nombre: 'Castillito', tipo_zona: 'residencial'});
CREATE (z3:Zona {nombre: 'Puerto Ordaz', tipo_zona: 'mixto'});
CREATE (z4:Zona {nombre: 'Villa Asia', tipo_zona: 'residencial'});
CREATE (z5:Zona {nombre: 'Los Olivos', tipo_zona: 'residencial'});
CREATE (z6:Zona {nombre: 'San Félix', tipo_zona: 'mixto'});
CREATE (z7:Zona {nombre: 'Unare', tipo_zona: 'comercial'});
CREATE (z8:Zona {nombre: 'Cauca', tipo_zona: 'residencial'});
CREATE (z9:Zona {nombre: 'Las Palmas', tipo_zona: 'residencial'});
CREATE (z10:Zona {nombre: 'Paseo Caroni', tipo_zona: 'comercial'});

// Conexión desde Centro Principal
MATCH (cd1:CentroDistribucion {nombre: 'Centro Principal'})
MATCH (z3:Zona {nombre: 'Puerto Ordaz'})
WHERE NOT EXISTS((cd1)-[:CONECTA]->(z3))
CREATE (cd1)-[:CONECTA {tiempo_minutos: 10, trafico_actual: 'medio', capacidad: 30, accesible:TRUE}]->(z3);

MATCH (cd1:CentroDistribucion {nombre: 'Centro Principal'})
MATCH (z10:Zona {nombre: 'Paseo Caroni'})
WHERE NOT EXISTS((cd1)-[:CONECTA]->(z10))
CREATE (cd1)-[:CONECTA {tiempo_minutos: 15, trafico_actual: 'bajo', capacidad: 25, accesible:TRUE}]->(z10);

// Conexión desde Centro Secundario
MATCH (cd2:CentroDistribucion {nombre: 'Centro Secundario'})
MATCH (z6:Zona {nombre: 'San Félix'})
WHERE NOT EXISTS((cd2)-[:CONECTA]->(z6))
CREATE (cd2)-[:CONECTA {tiempo_minutos: 8, trafico_actual: 'alto', capacidad: 20, accesible:TRUE}]->(z6);

MATCH (cd2:CentroDistribucion {nombre: 'Centro Secundario'})
MATCH (z7:Zona {nombre: 'Unare'})
WHERE NOT EXISTS((cd2)-[:CONECTA]->(z7))
CREATE (cd2)-[:CONECTA {tiempo_minutos: 12, trafico_actual: 'medio', capacidad: 25, accesible:TRUE}]->(z7);


// Conexiones comerciales principales
MATCH (z1:Zona {nombre: 'AltaVista'}), (z7:Zona {nombre: 'Unare'})
WHERE NOT EXISTS((z1)-[:CONECTA]-(z7))
CREATE (z1)-[:CONECTA {tiempo_minutos: 18, trafico_actual: 'alto', capacidad: 35, accesible:TRUE}]->(z7)
CREATE (z7)-[:CONECTA {tiempo_minutos: 18, trafico_actual: 'alto', capacidad: 35, accesible:TRUE}]->(z1);

MATCH (z1:Zona {nombre: 'AltaVista'}), (z10:Zona {nombre: 'Paseo Caroni'})
WHERE NOT EXISTS((z1)-[:CONECTA]-(z10))
CREATE (z1)-[:CONECTA {tiempo_minutos: 12, trafico_actual: 'medio', capacidad: 25, accesible:TRUE}]->(z10)
CREATE (z10)-[:CONECTA {tiempo_minutos: 12, trafico_actual: 'medio', capacidad: 25, accesible:TRUE}]->(z1);

MATCH (z7:Zona {nombre: 'Unare'}), (z10:Zona {nombre: 'Paseo Caroni'})
WHERE NOT EXISTS((z7)-[:CONECTA]-(z10))
CREATE (z7)-[:CONECTA {tiempo_minutos: 15, trafico_actual: 'alto', capacidad: 30, accesible:TRUE}]->(z10)
CREATE (z10)-[:CONECTA {tiempo_minutos: 15, trafico_actual: 'alto', capacidad: 30, accesible:TRUE}]->(z7);

// Red residencial Castillito-Villa Asia-Los Olivos
MATCH (z2:Zona {nombre: 'Castillito'}), (z4:Zona {nombre: 'Villa Asia'})
WHERE NOT EXISTS((z2)-[:CONECTA]-(z4))
CREATE (z2)-[:CONECTA {tiempo_minutos: 7, trafico_actual: 'bajo', capacidad: 15, accesible:TRUE}]->(z4)
CREATE (z4)-[:CONECTA {tiempo_minutos: 7, trafico_actual: 'bajo', capacidad: 15, accesible:TRUE}]->(z2);

MATCH (z2:Zona {nombre: 'Castillito'}), (z5:Zona {nombre: 'Los Olivos'})
WHERE NOT EXISTS((z2)-[:CONECTA]-(z5))
CREATE (z2)-[:CONECTA {tiempo_minutos: 10, trafico_actual: 'bajo', capacidad: 20, accesible:TRUE}]->(z5)
CREATE (z5)-[:CONECTA {tiempo_minutos: 10, trafico_actual: 'bajo', capacidad: 20, accesible:TRUE}]->(z2);

MATCH (z4:Zona {nombre: 'Villa Asia'}), (z5:Zona {nombre: 'Los Olivos'})
WHERE NOT EXISTS((z4)-[:CONECTA]-(z5))
CREATE (z4)-[:CONECTA {tiempo_minutos: 5, trafico_actual: 'bajo', capacidad: 15, accesible:TRUE}]->(z5)
CREATE (z5)-[:CONECTA {tiempo_minutos: 5, trafico_actual: 'bajo', capacidad: 15, accesible:TRUE}]->(z4);


MATCH (z3:Zona {nombre: 'Puerto Ordaz'}), (z6:Zona {nombre: 'San Félix'})
WHERE NOT EXISTS((z3)-[:CONECTA]-(z6))
CREATE (z3)-[:CONECTA {tiempo_minutos: 20, trafico_actual: 'medio', capacidad: 25, accesible:TRUE}]->(z6);

MATCH (z3:Zona {nombre: 'Puerto Ordaz'}), (z2:Zona {nombre: 'Castillito'})
WHERE NOT EXISTS((z3)-[:CONECTA]-(z2))
CREATE (z3)-[:CONECTA {tiempo_minutos: 11, trafico_actual: 'bajo', capacidad: 6, accesible:TRUE}]->(z2)
CREATE (z2)-[:CONECTA {tiempo_minutos: 11, trafico_actual: 'bajo', capacidad: 6, accesible:TRUE}]->(z3);


MATCH (z3:Zona {nombre: 'Puerto Ordaz'}), (z6:Zona {nombre: 'AltaVista'})
MERGE (z6)-[r:CONECTA]->(z3) // Intenta encontrar esta relación, si no existe, la crea
ON CREATE SET r.tiempo_minutos = 40, r.trafico_actual = 'alto', r.capacidad = 30, r.accesible=TRUE;


