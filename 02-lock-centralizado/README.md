# Solución 1: Lock Centralizado

Esta implementación utiliza un coordinador centralizado para manejar los bloqueos distribuidos, evitando las condiciones de carrera en el sistema de reservas de asientos.

## Arquitectura

```
Frontend (Astro) 
    ↓
Nginx Load Balancer (Puerto 80)
    ↓
Reservation Servers (Puertos 8081, 8082, 8083)
    ↓
Lock Coordinator (Puerto 8080)
    ↓
MongoDB (Puerto 27017)
```

## Componentes

### 1. Lock Coordinator (`coordinator/`)
- **Puerto**: 8080
- **Función**: Maneja todos los bloqueos distribuidos
- **Endpoints**:
  - `POST /acquire` - Adquirir un bloqueo
  - `POST /release` - Liberar un bloqueo
  - `GET /status/{resource}` - Estado de un bloqueo
  - `GET /health` - Health check

### 2. Reservation Servers (`server/`)
- **Puertos**: 8081, 8082, 8083
- **Función**: Manejan las reservas de asientos
- **Endpoints**:
  - `GET /asientos` - Obtener todos los asientos
  - `POST /reservar` - Reservar un asiento
  - `POST /liberar` - Liberar un asiento
  - `GET /health` - Health check

### 3. MongoDB
- **Puerto**: 27017
- **Bases de datos**:
  - `locks_db.locks` - Almacena los bloqueos activos
  - `reservations_db.seats` - Almacena el estado de los asientos

### 4. Nginx Load Balancer
- **Puerto**: 80
- **Función**: Distribuye las peticiones entre los servidores de reservas

## Cómo funciona

1. **Solicitud de reserva**: El cliente envía una solicitud al frontend
2. **Load balancing**: Nginx distribuye la petición a uno de los servidores
3. **Adquisición de bloqueo**: El servidor solicita un bloqueo al coordinador
4. **Operación atómica**: Si obtiene el bloqueo, realiza la reserva
5. **Liberación**: El servidor libera el bloqueo automáticamente

## Instalación y Ejecución

### Prerrequisitos
- Docker y Docker Compose
- Go 1.21+ (para desarrollo)

### Ejecutar el sistema

1. **Construir y ejecutar todos los servicios**:
```bash
cd 02-lock-centralizado
docker-compose up --build
```

2. **Verificar que todos los servicios estén funcionando**:
```bash
# Coordinator
curl http://localhost:8080/health

# Servers
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8083/health

# Load balancer
curl http://localhost/health
```

3. **Probar el sistema**:
```bash
# Obtener asientos
curl http://localhost/asientos

# Reservar asiento
curl -X POST http://localhost/reservar \
  -H "Content-Type: application/json" \
  -d '{"numero": 1, "cliente": "Juan Pérez"}'

# Liberar asiento
curl -X POST http://localhost/liberar \
  -H "Content-Type: application/json" \
  -d '{"numero": 1}'
```

## Configuración del Frontend

El frontend debe apuntar a `http://localhost` (puerto 80) para usar el load balancer, o directamente a los servidores individuales:
- Server 1: `http://localhost:8081`
- Server 2: `http://localhost:8082`
- Server 3: `http://localhost:8083`

## Ventajas de esta solución

1. **Consistencia**: El coordinador centralizado garantiza que no haya condiciones de carrera
2. **Escalabilidad**: Se pueden agregar más servidores de reservas fácilmente
3. **Tolerancia a fallos**: Si un servidor falla, los otros continúan funcionando
4. **Persistencia**: Los datos se almacenan en MongoDB
5. **Load balancing**: Nginx distribuye la carga entre servidores

## Desventajas

1. **Punto único de falla**: El coordinador es crítico para el sistema
2. **Latencia**: Cada operación requiere comunicación con el coordinador
3. **Complejidad**: Más componentes que mantener

## Monitoreo

Todos los servicios exponen un endpoint `/health` para monitoreo:
- Coordinator: http://localhost:8080/health
- Servers: http://localhost:808[1-3]/health
- Load balancer: http://localhost/health

## Logs

Para ver los logs de todos los servicios:
```bash
docker-compose logs -f
```

Para ver logs de un servicio específico:
```bash
docker-compose logs -f coordinator
docker-compose logs -f server1
```



### 1. La Arquitectura: El Restaurante Organizado
Imagina el sistema como un restaurante muy concurrido:

- Tú (El Cliente) : Eres el usuario en la página web, interactuando con la interfaz que ves en `lock-centralizado.astro` .
- El Recepcionista (Load Balancer - Nginx en el puerto 80) : En lugar de ir directamente a cualquier camarero, ahora hablas únicamente con un recepcionista en la entrada. Este es el Load Balancer . Su trabajo es tomar tu pedido (la reserva) y encontrar un camarero que no esté ocupado para atenderte.
- Los Camareros (Servidores de Reserva - Puertos 8081, 8082, 8083) : Son los que hacen el trabajo de procesar la reserva. El recepcionista le pasa tu pedido a uno de ellos de forma balanceada.
- El Gerente con el Libro de Reservas (Coordinador de Locks - Puerto 8080) : Aquí está la clave de la solución. Antes de que un camarero pueda confirmarte una mesa (reservar un asiento), debe pedirle permiso al gerente . El gerente tiene el único libro maestro de reservas ( locks ) y se asegura de que dos camareros no intenten darle la misma mesa a dos clientes diferentes al mismo tiempo.
Esta es exactamente la arquitectura que se describe en el `README.md` que has señalado. El flujo es:

1. 1.
   Cliente → Load Balancer : Tu navegador envía la solicitud de reserva al puerto 80.
2. 2.
   Load Balancer → Servidor : El Load Balancer elige uno de los servidores (ej. 8081) y le reenvía la solicitud.
3. 3.
   Servidor → Coordinador : El servidor 8081, antes de tocar la base de datos, le pide al Coordinador (puerto 8080) un "lock" (un permiso exclusivo) para el asiento que quieres.
4. 4.
   Coordinador → Servidor : Si nadie más tiene ese permiso, el Coordinador se lo concede.
5. 5.
   Servidor → Base de Datos : Con el permiso en mano, el servidor reserva el asiento.
6. 6.
   Servidor → Coordinador : El servidor le avisa al Coordinador que ha terminado, liberando el permiso.
7. 7.
   La respuesta exitosa viaja de vuelta hasta tu navegador.
### 2. ¿Por qué ya no puedes seleccionar los puertos 8081, 8082 y 8083?
Esta es la parte más importante y la razón del cambio para que el sistema sea correcto.

Antes, cada "camarero" (servidor en 8081, 8082, etc.) tenía su propia política de "a quién atender" (lo que se conoce como CORS). Podías hablar con cualquiera de ellos directamente. El problema es que no hablaban entre sí antes de confirmar una reserva, y por eso se producían las reservas duplicadas.