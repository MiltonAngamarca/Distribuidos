# 🚨 PROBLEMA: Sistema de Reservas con Race Conditions

## 📋 Descripción del Problema

Este directorio contiene una implementación **intencionalmente incorrecta** de un sistema de reservas distribuido que demuestra los problemas de **race conditions** en sistemas concurrentes.

### ❌ ¿Qué está mal?

El sistema permite que múltiples servidores procesen reservas del mismo asiento **simultáneamente** sin ningún mecanismo de sincronización, lo que resulta en:

- **Doble reserva**: El mismo asiento puede ser reservado por múltiples clientes
- **Estado inconsistente**: Diferentes servidores pueden tener estados diferentes
- **Pérdida de datos**: Las reservas pueden sobrescribirse mutuamente

---

## 🏗️ Arquitectura del Sistema

```
Cliente A ──┐
Cliente B ──┼──► Load Balancer ──┐
Cliente C ──┘                    ├──► Servidor 1 (Puerto 8081)
                                 ├──► Servidor 2 (Puerto 8082)
                                 └──► Servidor 3 (Puerto 8083)
```

### Componentes:

1. **3 Servidores Go** - Cada uno maneja su propio estado independiente
2. **Load Balancer (nginx)** - Distribuye peticiones entre servidores
3. **Scripts de Prueba** - Para demostrar el race condition

---

## 🔧 Estructura de Archivos

```
01-problema/
├── README.md                    # Este archivo
├── main.go                      # Servidor con race condition
├── models/
│   └── asiento.go              # Modelo de datos (sin mutex)
├── Dockerfile                   # Imagen del servidor
├── docker-compose.yml          # Orquestación de 3 servidores
├── nginx.conf                   # Configuración del load balancer
├── test-race-condition.sh       # Script de prueba (Linux/Mac)
└── test-race-condition.ps1     # Script de prueba (Windows)
```

---

## 🚀 Cómo Ejecutar

### Paso 1: Levantar los Servidores

```bash
# Construir y levantar todos los servicios
docker-compose up --build

# O en modo detached
docker-compose up --build -d
```

Esto levantará:
- **Servidor 1**: http://localhost:8081
- **Servidor 2**: http://localhost:8082  
- **Servidor 3**: http://localhost:8083
- **Load Balancer**: http://localhost:8080

### Paso 2: Verificar que Funcionen

```bash
# Verificar estado de los servidores
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8083/health

# Ver asientos disponibles
curl http://localhost:8081/asientos
```

### Paso 3: Demostrar el Race Condition

#### En Linux/Mac:
```bash
# Hacer ejecutable el script
chmod +x test-race-condition.sh

# Ejecutar prueba
./test-race-condition.sh
```

#### En Windows:
```powershell
# Ejecutar script de PowerShell
.\test-race-condition.ps1
```

---

## 🎯 Demostración del Problema

### Escenario de Race Condition:

1. **Cliente A** envía petición al **Servidor 1** para reservar asiento #5
2. **Cliente B** envía petición al **Servidor 2** para reservar asiento #5  
3. **Cliente C** envía petición al **Servidor 3** para reservar asiento #5

**Todas las peticiones llegan casi simultáneamente**

### ❌ Resultado Incorrecto:

```
Servidor 1: Asiento 5 - RESERVADO por Cliente A
Servidor 2: Asiento 5 - RESERVADO por Cliente B  
Servidor 3: Asiento 5 - RESERVADO por Cliente C
```

**¡El mismo asiento está "reservado" 3 veces!**

---

## 🔍 Análisis del Código Problemático

### El Race Condition en `models/asiento.go`:

```go
// PROBLEMA: Check-then-act sin sincronización
func (s *SistemaReservas) ReservarAsiento(numero int, cliente string) error {
    asiento, existe := s.Asientos[numero]
    if !existe {
        return &ReservaError{...}
    }
    
    // ❌ RACE CONDITION AQUÍ
    if asiento.Disponible {                    // 1. CHECK
        time.Sleep(100 * time.Millisecond)     // 2. Simula latencia
        asiento.Disponible = false             // 3. ACT
        asiento.Cliente = cliente
        return nil
    }
    
    return &ReservaError{...}
}
```

### ¿Por qué falla?

1. **Thread A** verifica `asiento.Disponible == true` ✅
2. **Thread B** verifica `asiento.Disponible == true` ✅ (aún no cambió)
3. **Thread C** verifica `asiento.Disponible == true` ✅ (aún no cambió)
4. **Thread A** cambia `asiento.Disponible = false`
5. **Thread B** cambia `asiento.Disponible = false` (sobrescribe A)
6. **Thread C** cambia `asiento.Disponible = false` (sobrescribe B)

**Resultado**: Todos creen que reservaron exitosamente.

---

## 📊 API Endpoints

### GET `/health`
Verifica el estado del servidor
```json
{
  "status": "ok",
  "servidor": "servidor-1",
  "timestamp": "2024-01-20T10:30:00Z"
}
```

### GET `/asientos`
Lista todos los asientos
```json
{
  "servidor": "servidor-1",
  "asientos": {
    "1": {
      "numero": 1,
      "disponible": true,
      "servidor_id": "servidor-1"
    }
  }
}
```

### POST `/reservar`
Reserva un asiento
```json
// Request
{
  "numero": 5,
  "cliente": "Juan Pérez"
}

// Response (éxito)
{
  "success": true,
  "message": "Asiento reservado exitosamente",
  "asiento": {
    "numero": 5,
    "disponible": false,
    "cliente": "Juan Pérez",
    "fecha_reserva": "2024-01-20T10:30:00Z"
  }
}
```

### POST `/liberar`
Libera un asiento reservado
```json
// Request
{
  "numero": 5
}

// Response
{
  "success": true,
  "message": "Asiento liberado exitosamente"
}
```

### GET `/estado`
Estado general del sistema
```json
{
  "servidor_id": "servidor-1",
  "total_asientos": 50,
  "disponibles": 45,
  "reservados": 5,
  "ultima_actualizacion": "2024-01-20T10:30:00Z"
}
```

---

## 🧪 Scripts de Prueba

### Funcionalidades del Script:

1. **Verificar Servidores** - Comprobar que todos estén activos
2. **Resetear Sistema** - Limpiar todas las reservas
3. **Prueba Individual** - Demostrar race condition en un asiento
4. **Pruebas Múltiples** - Ejecutar N pruebas y calcular estadísticas
5. **Estado General** - Ver resumen de todos los asientos

### Ejemplo de Salida:

```
🚀 INICIANDO PRUEBA DE RACE CONDITION
Asiento objetivo: 5

📋 Estado inicial:
  Servidor 1: DISPONIBLE
  Servidor 2: DISPONIBLE  
  Servidor 3: DISPONIBLE

⚡ Enviando peticiones concurrentes...

✅ [Cliente-A] Reserva exitosa en servidor puerto 8081
✅ [Cliente-B] Reserva exitosa en servidor puerto 8082
✅ [Cliente-C] Reserva exitosa en servidor puerto 8083

📋 Estado final:
  Servidor 1: RESERVADO por 'Cliente-A'
  Servidor 2: RESERVADO por 'Cliente-B'
  Servidor 3: RESERVADO por 'Cliente-C'

🚨 RACE CONDITION DETECTADO!
   El asiento 5 está reservado en 3 servidores
   Esto demuestra el problema de concurrencia
```

---

## 🎓 Conceptos Demostrados

### 1. Race Condition
- **Definición**: Cuando el resultado depende del timing de eventos concurrentes
- **Causa**: Acceso no sincronizado a recursos compartidos
- **Efecto**: Comportamiento impredecible y datos corruptos

### 2. Check-Then-Act
- **Patrón problemático**: Verificar condición y luego actuar
- **Problema**: La condición puede cambiar entre check y act
- **Solución**: Operaciones atómicas

### 3. Estado Distribuido
- **Problema**: Múltiples copias del mismo estado
- **Desafío**: Mantener consistencia entre copias
- **Necesidad**: Mecanismos de sincronización

---

## ⚠️ Limitaciones Conocidas

1. **Sin Persistencia**: Los datos se pierden al reiniciar
2. **Sin Autenticación**: Cualquiera puede hacer reservas
3. **Sin Validación**: Datos de entrada mínimamente validados
4. **Sin Logging Distribuido**: Logs solo locales
5. **Sin Métricas**: No hay monitoreo de performance

---

## 🔄 Próximos Pasos

Este problema será resuelto en las siguientes carpetas:

1. **`02-lock-centralizado/`** - Solución con coordinador central
2. **`03-lock-distribuido/`** - Algoritmo Ricart-Agrawala  
3. **`04-two-phase-commit/`** - Transacciones distribuidas

---

## 🛠️ Comandos Útiles

```bash
# Ver logs en tiempo real
docker-compose logs -f

# Reiniciar solo un servicio
docker-compose restart servidor-1

# Escalar servidores (agregar más instancias)
docker-compose up --scale servidor-1=2

# Parar todo
docker-compose down

# Limpiar volúmenes
docker-compose down -v

# Reconstruir imágenes
docker-compose build --no-cache
```

---

## 📚 Referencias

- [Go Race Detector](https://golang.org/doc/articles/race_detector.html)
- [Distributed Systems Concepts](https://en.wikipedia.org/wiki/Distributed_computing)
- [CAP Theorem](https://en.wikipedia.org/wiki/CAP_theorem)
- [Consistency Models](https://en.wikipedia.org/wiki/Consistency_model)

---

## 🤝 Contribuir

Este es un proyecto educativo. Si encuentras mejoras o errores:

1. Abre un issue describiendo el problema
2. Propón una solución
3. Mantén el enfoque educativo

**Recuerda**: El objetivo es demostrar el problema, no solucionarlo aquí.