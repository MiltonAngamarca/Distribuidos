package models

import (
	"time"
)

// Asiento representa un asiento en el sistema de reservas
type Asiento struct {
	Numero      int       `json:"numero"`
	Disponible  bool      `json:"disponible"`
	Cliente     string    `json:"cliente,omitempty"`
	FechaReserva *time.Time `json:"fecha_reserva,omitempty"`
	ServidorID  string    `json:"servidor_id"`
}

// SistemaReservas maneja el estado de los asientos
// NOTA: Esta implementación tiene race conditions intencionalmente
type SistemaReservas struct {
	Asientos   map[int]*Asiento `json:"asientos"`
	ServidorID string           `json:"servidor_id"`
	// NO usamos mutex aquí para demostrar el problema
	// mutex      sync.RWMutex
}

// NewSistemaReservas crea un nuevo sistema de reservas
func NewSistemaReservas(servidorID string, totalAsientos int) *SistemaReservas {
	asientos := make(map[int]*Asiento)
	
	// Inicializar asientos disponibles
	for i := 1; i <= totalAsientos; i++ {
		asientos[i] = &Asiento{
			Numero:     i,
			Disponible: true,
			ServidorID: servidorID,
		}
	}
	
	return &SistemaReservas{
		Asientos:   asientos,
		ServidorID: servidorID,
	}
}

// ReservarAsiento intenta reservar un asiento
// PROBLEMA: Esta función tiene race condition
func (s *SistemaReservas) ReservarAsiento(numero int, cliente string) error {
	// Verificar si el asiento existe
	asiento, existe := s.Asientos[numero]
	if !existe {
		return &ReservaError{
			Codigo:  "ASIENTO_NO_EXISTE",
			Mensaje: "El asiento no existe",
		}
	}
	
	// RACE CONDITION: Check-then-act sin sincronización
	if asiento.Disponible {
		// Simular latencia de red/procesamiento
		time.Sleep(100 * time.Millisecond)
		
		// Cambiar estado del asiento
		now := time.Now()
		asiento.Disponible = false
		asiento.Cliente = cliente
		asiento.FechaReserva = &now
		asiento.ServidorID = s.ServidorID
		
		return nil
	}
	
	return &ReservaError{
		Codigo:  "ASIENTO_NO_DISPONIBLE",
		Mensaje: "El asiento ya está reservado",
	}
}

// LiberarAsiento libera un asiento reservado
func (s *SistemaReservas) LiberarAsiento(numero int) error {
	asiento, existe := s.Asientos[numero]
	if !existe {
		return &ReservaError{
			Codigo:  "ASIENTO_NO_EXISTE",
			Mensaje: "El asiento no existe",
		}
	}
	
	if asiento.Disponible {
		return &ReservaError{
			Codigo:  "ASIENTO_YA_LIBRE",
			Mensaje: "El asiento ya está libre",
		}
	}
	
	// Liberar asiento
	asiento.Disponible = true
	asiento.Cliente = ""
	asiento.FechaReserva = nil
	
	return nil
}

// ObtenerAsiento devuelve información de un asiento específico
func (s *SistemaReservas) ObtenerAsiento(numero int) (*Asiento, error) {
	asiento, existe := s.Asientos[numero]
	if !existe {
		return nil, &ReservaError{
			Codigo:  "ASIENTO_NO_EXISTE",
			Mensaje: "El asiento no existe",
		}
	}
	
	// Crear copia para evitar modificaciones externas
	copia := *asiento
	return &copia, nil
}

// ObtenerTodosLosAsientos devuelve todos los asientos
func (s *SistemaReservas) ObtenerTodosLosAsientos() map[int]*Asiento {
	// Crear copia del mapa para evitar modificaciones externas
	copia := make(map[int]*Asiento)
	for numero, asiento := range s.Asientos {
		asientoCopia := *asiento
		copia[numero] = &asientoCopia
	}
	return copia
}

// ContarDisponibles cuenta los asientos disponibles
func (s *SistemaReservas) ContarDisponibles() int {
	contador := 0
	for _, asiento := range s.Asientos {
		if asiento.Disponible {
			contador++
		}
	}
	return contador
}

// ContarReservados cuenta los asientos reservados
func (s *SistemaReservas) ContarReservados() int {
	contador := 0
	for _, asiento := range s.Asientos {
		if !asiento.Disponible {
			contador++
		}
	}
	return contador
}

// ReservaError representa un error en el sistema de reservas
type ReservaError struct {
	Codigo  string `json:"codigo"`
	Mensaje string `json:"mensaje"`
}

func (e *ReservaError) Error() string {
	return e.Mensaje
}

// EstadoSistema devuelve el estado actual del sistema
type EstadoSistema struct {
	ServidorID       string `json:"servidor_id"`
	TotalAsientos    int    `json:"total_asientos"`
	Disponibles      int    `json:"disponibles"`
	Reservados       int    `json:"reservados"`
	UltimaActualizacion time.Time `json:"ultima_actualizacion"`
}

// ObtenerEstado devuelve el estado actual del sistema
func (s *SistemaReservas) ObtenerEstado() *EstadoSistema {
	return &EstadoSistema{
		ServidorID:       s.ServidorID,
		TotalAsientos:    len(s.Asientos),
		Disponibles:      s.ContarDisponibles(),
		Reservados:       s.ContarReservados(),
		UltimaActualizacion: time.Now(),
	}
}