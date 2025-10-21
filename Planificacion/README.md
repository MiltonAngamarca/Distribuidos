# 🎬 SISTEMA DE RESERVAS DISTRIBUIDO
## Proyecto Educativo - Sincronización Distribuida en GO

---

## 📚 ÍNDICE DEL PROYECTO

### PARTE 1: PROBLEMA (Implementación Incorrecta)
- ✅ `01-problema/` - Sistema con race conditions
- ✅ Demostración del fallo de concurrencia
- ✅ Docker Compose para simular ambiente distribuido

### PARTE 2: SOLUCIÓN 1 - Lock Centralizado
- ✅ `02-lock-centralizado/` - Coordinador de locks
- ✅ Diagrama de arquitectura
- ✅ Ventajas y desventajas

### PARTE 3: SOLUCIÓN 2 - Lock Distribuido (Ricart-Agrawala)
- ✅ `03-lock-distribuido/` - Algoritmo sin coordinador
- ✅ Diagrama de arquitectura
- ✅ Manejo de consenso

### PARTE 4: SOLUCIÓN 3 - Protocolo 2PC
- ✅ `04-two-phase-commit/` - Transacciones distribuidas
- ✅ Diagrama de arquitectura
- ✅ Manejo de fallos

### PARTE 5: Frontend React
- ✅ `frontend/` - Dashboard de monitoreo
- ✅ React + Tailwind CSS
- ✅ Visualización en tiempo real

### PARTE 6: Diagramas y Documentación
- ✅ `docs/` - Diagramas HTML interactivos
- ✅ Comparativa de soluciones
- ✅ Análisis de trade-offs

---

## 🏗️ ESTRUCTURA DEL PROYECTO

```
distributed-reservations/
│
├── README.md                          # Este archivo
├── docker-compose.yml                 # Orquestación de todos los servicios
│
├── 01-problema/                       # ❌ IMPLEMENTACIÓN INCORRECTA
│   ├── README.md                      # Explicación del problema
│   ├── Dockerfile
│   ├── main.go                        # Servidor con race condition
│   ├── models/asiento.go
│   └── docker-compose.yml
│
├── 02-lock-centralizado/              # ✅ SOLUCIÓN 1
│   ├── README.md                      # Documentación
│   ├── Dockerfile
│   ├── coordinator/
│   │   ├── main.go                    # Coordinador de locks
│   │   └── lock_manager.go
│   ├── server/
│   │   ├── main.go                    # Servidor de reservas
│   │   └── handlers.go
│   └── docker-compose.yml
│
├── 03-lock-distribuido/               # ✅ SOLUCIÓN 2
│   ├── README.md
│   ├── Dockerfile
│   ├── server/
│   │   ├── main.go                    # Algoritmo Ricart-Agrawala
│   │   ├── ricart_agrawala.go
│   │   └── lamport_clock.go
│   └── docker-compose.yml
│
├── 04-two-phase-commit/               # ✅ SOLUCIÓN 3
│   ├── README.md
│   ├── Dockerfile
│   ├── coordinator/
│   │   ├── main.go                    # Coordinador 2PC
│   │   └── two_phase_commit.go
│   ├── participant/
│   │   ├── main.go                    # Participante
│   │   └── transaction_manager.go
│   └── docker-compose.yml
│
├── frontend/                          # 🎨 REACT APP
│   ├── README.md
│   ├── Dockerfile
│   ├── package.json
│   ├── public/
│   ├── src/
│   │   ├── App.jsx
│   │   ├── components/
│   │   │   ├── ReservationPanel.jsx  # Panel de reservas
│   │   │   ├── ServerStatus.jsx      # Estado de servidores
│   │   │   └── ConflictLog.jsx       # Log de conflictos
│   │   └── services/
│   │       └── api.js
│   └── tailwind.config.js
│
├── docs/                              # 📊 DOCUMENTACIÓN
│   ├── index.html                     # Página principal
│   ├── problema.html                  # Diagrama del problema
│   ├── solucion-1.html                # Diagrama lock centralizado
│   ├── solucion-2.html                # Diagrama lock distribuido
│   ├── solucion-3.html                # Diagrama 2PC
│   ├── comparativa.html               # Tabla comparativa
│   └── css/
│       └── styles.css
│
└── shared/                            # 🔧 CÓDIGO COMPARTIDO
    ├── models/
    │   └── asiento.go
    └── utils/
        └── logger.go
```

---

## 🚀 ORDEN DE IMPLEMENTACIÓN

### FASE 1: Problema y Fundamentos
```bash
# 1. Implementar sistema con race condition
cd 01-problema/
docker-compose up

# 2. Demostrar el fallo
# - Levantar 3 servidores
# - Simular peticiones concurrentes
# - Mostrar doble reserva
```

### FASE 2: Soluciones Progresivas
```bash
# 3. Implementar lock centralizado
cd 02-lock-centralizado/
docker-compose up

# 4. Implementar lock distribuido
cd 03-lock-distribuido/
docker-compose up

# 5. Implementar 2PC
cd 04-two-phase-commit/
docker-compose up
```

### FASE 3: Frontend y Visualización
```bash
# 6. Levantar frontend
cd frontend/
npm install
npm run dev

# 7. Ver diagramas
cd docs/
python -m http.server 8080
```

---

## 🎯 OBJETIVOS PEDAGÓGICOS

### 1️⃣ PROBLEMA (01-problema)
**Demostrar:**
- ❌ Race conditions en sistemas distribuidos
- ❌ Doble reserva del mismo asiento
- ❌ Inconsistencia entre servidores

**Código clave:**
```go
// SIN SINCRONIZACIÓN - INCORRECTO
func ReservarAsiento(numero int, cliente string) error {
    if asientos[numero].Disponible {  // ← PROBLEMA: Check-then-act
        time.Sleep(100 * time.Millisecond) // Simula latencia
        asientos[numero].Disponible = false
        asientos[numero].Cliente = cliente
        return nil
    }
    return errors.New("asiento no disponible")
}
```

### 2️⃣ SOLUCIÓN 1: Lock Centralizado
**Enseñar:**
- ✅ Coordinador central de locks
- ✅ Protocolo request-grant-release
- ⚠️ Single point of failure
- ⚠️ Cuello de botella

**Arquitectura:**
```
Cliente 1 ──┐
Cliente 2 ──┼──► Coordinador ──► Servidor A
Cliente 3 ──┘      (Locks)    └──► Servidor B
                                └──► Servidor C
```

### 3️⃣ SOLUCIÓN 2: Lock Distribuido
**Enseñar:**
- ✅ Algoritmo Ricart-Agrawala
- ✅ Consenso entre todos los nodos
- ✅ Sin punto único de fallo
- ⚠️ Requiere N mensajes (todos a todos)

**Arquitectura:**
```
Servidor A ←──┐
    ↕         ├──► Consenso entre todos
Servidor B ←──┤
    ↕         │
Servidor C ←──┘
```

### 4️⃣ SOLUCIÓN 3: Two-Phase Commit
**Enseñar:**
- ✅ Transacciones distribuidas
- ✅ Fase PREPARE + Fase COMMIT
- ✅ Rollback automático
- ⚠️ Bloqueo si coordinador falla

**Arquitectura:**
```
                Coordinador
                    │
        ┌───────────┼───────────┐
        ↓           ↓           ↓
  Participante  Participante  Participante
      A             B             C
  (Asiento)    (Parking)      (Puntos)
```

---

## 🔧 TECNOLOGÍAS

### Backend
- **Lenguaje:** Go 1.21+
- **Framework:** net/http estándar
- **Concurrencia:** Goroutines + Channels
- **Serialización:** encoding/json

### Frontend
- **Framework:** React 18
- **Estilos:** Tailwind CSS
- **HTTP:** Axios
- **Estado:** React Hooks

### Infraestructura
- **Contenedores:** Docker + Docker Compose
- **Networking:** Bridge network
- **Persistencia:** En memoria (educativo)

---

## 📊 MÉTRICAS A COMPARAR

| Solución | Throughput | Latencia | Disponibilidad | Complejidad |
|----------|------------|----------|----------------|-------------|
| Problema | ⚠️ Alto | ✅ Baja | ✅ Alta | ✅ Baja |
| Lock Central | ⚠️ Medio | ⚠️ Media | ❌ Baja (SPOF) | ✅ Baja |
| Lock Distrib | ❌ Bajo | ❌ Alta | ✅ Alta | ❌ Alta |
| 2PC | ⚠️ Medio | ⚠️ Media | ⚠️ Media | ⚠️ Media |

---

## 🎓 CONCEPTOS CUBIERTOS

### Sincronización
- [x] Race conditions
- [x] Exclusión mutua distribuida
- [x] Algoritmos de consenso
- [x] Lamport timestamps

### Transacciones
- [x] ACID en sistemas distribuidos
- [x] Two-Phase Commit (2PC)
- [x] Write-Ahead Log (WAL)
- [x] Rollback y recuperación

### Fallos
- [x] Particiones de red
- [x] Fallo de nodos
- [x] Timeout y recovery
- [x] Deadlock distribuido

### Teoría
- [x] Teorema CAP
- [x] Consistencia vs Disponibilidad
- [x] Linearizability
- [x] Eventual consistency

---

## 🚦 PLAN DE DESARROLLO

### ✅ PASO 1: Estructura Base (TÚ ESTÁS AQUÍ)
- [x] README.md con plan completo
- [ ] Estructura de carpetas
- [ ] Docker compose base

### 🔄 PASO 2: Implementación del Problema
- [ ] Servidor Go con race condition
- [ ] 3 instancias en Docker
- [ ] Script de prueba de concurrencia
- [ ] Diagrama HTML del problema

### 🔄 PASO 3: Solución 1 - Lock Centralizado
- [ ] Coordinador de locks
- [ ] Servidores con cliente de locks
- [ ] Diagrama de arquitectura
- [ ] Comparativa con el problema

### 🔄 PASO 4: Solución 2 - Lock Distribuido
- [ ] Algoritmo Ricart-Agrawala
- [ ] Relojes de Lamport
- [ ] Manejo de mensajes
- [ ] Diagrama de secuencia

### 🔄 PASO 5: Solución 3 - 2PC
- [ ] Coordinador 2PC
- [ ] Participantes con WAL
- [ ] Manejo de fallos
- [ ] Diagrama de fases

### 🔄 PASO 6: Frontend
- [ ] Dashboard React
- [ ] Componentes de visualización
- [ ] Conexión con backends
- [ ] Estilos Tailwind

### 🔄 PASO 7: Documentación Final
- [ ] Diagramas interactivos HTML
- [ ] Página comparativa
- [ ] Guía de uso
- [ ] Video demo

---

## 📝 COMANDOS RÁPIDOS

```bash
# Levantar problema
make problema

# Levantar solución 1
make solucion1

# Levantar solución 2
make solucion2

# Levantar solución 3
make solucion3

# Levantar frontend
make frontend

# Ver documentación
make docs

# Limpiar todo
make clean

# Prueba de carga
make stress-test
```

---

## 👨‍🏫 FLUJO DE LA CLASE

### 1. Introducción (10 min)
- Mostrar el problema en `01-problema/`
- Ejecutar test que causa doble reserva
- Explicar por qué falla

### 2. Solución 1 (15 min)
- Implementar lock centralizado
- Mostrar que funciona
- Discutir limitaciones (SPOF)

### 3. Solución 2 (15 min)
- Implementar algoritmo distribuido
- Explicar mensajes entre nodos
- Discutir overhead de comunicación

### 4. Solución 3 (20 min)
- Implementar 2PC completo
- Simular fallo y rollback
- Explicar garantías ACID

### 5. Comparativa (10 min)
- Ver diagramas en `docs/`
- Analizar trade-offs
- Discutir casos de uso reales

---

## 🎯 SIGUIENTE PASO

**Ahora vamos a implementar PASO 2:**
1. Crear el problema (servidor con race condition)
2. Docker compose con 3 servidores
3. Script para demostrar el fallo
4. Diagrama HTML explicativo

¿Comenzamos con el Paso 2? 🚀