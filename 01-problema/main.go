package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"problema-reservas/models"
)

var (
	sistema    *models.SistemaReservas
	servidorID string
	puerto     string
)

func init() {
	// Obtener ID del servidor desde variable de entorno
	servidorID = os.Getenv("SERVIDOR_ID")
	if servidorID == "" {
		servidorID = "servidor-1"
	}

	// Obtener puerto desde variable de entorno
	puerto = os.Getenv("PUERTO")
	if puerto == "" {
		puerto = "8080"
	}

	// Inicializar sistema con 50 asientos
	sistema = models.NewSistemaReservas(servidorID, 50)
	
	log.Printf("🚀 Servidor %s iniciado en puerto %s", servidorID, puerto)
	log.Printf("⚠️  ADVERTENCIA: Este servidor tiene race conditions intencionalmente")
}

func main() {
	// Configurar rutas
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/asientos", asientosHandler)
	http.HandleFunc("/asiento/", asientoHandler)
	http.HandleFunc("/reservar", reservarHandler)
	http.HandleFunc("/liberar", liberarHandler)
	http.HandleFunc("/estado", estadoHandler)
	http.HandleFunc("/reset", resetHandler)

	// Configurar CORS para permitir requests desde el frontend
	http.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		
		// Rutear a los handlers apropiados
		switch r.URL.Path {
		case "/api/asientos":
			asientosHandler(w, r)
		case "/api/estado":
			estadoHandler(w, r)
		case "/api/reservar":
			reservarHandler(w, r)
		case "/api/liberar":
			liberarHandler(w, r)
		default:
			http.NotFound(w, r)
		}
	})

	// Iniciar servidor
	log.Printf("🌐 Servidor escuchando en http://localhost:%s", puerto)
	log.Printf("📊 Endpoints disponibles:")
	log.Printf("   GET  /health        - Estado del servidor")
	log.Printf("   GET  /asientos      - Lista todos los asientos")
	log.Printf("   GET  /asiento/{id}  - Información de un asiento")
	log.Printf("   POST /reservar      - Reservar un asiento")
	log.Printf("   POST /liberar       - Liberar un asiento")
	log.Printf("   GET  /estado        - Estado del sistema")
	log.Printf("   POST /reset         - Reiniciar sistema")
	
	if err := http.ListenAndServe(":"+puerto, nil); err != nil {
		log.Fatal("❌ Error al iniciar servidor:", err)
	}
}

// enableCORS habilita CORS para requests del frontend
func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// homeHandler maneja la ruta raíz
func homeHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	response := map[string]interface{}{
		"servidor":    servidorID,
		"mensaje":     "Sistema de Reservas - Problema con Race Conditions",
		"advertencia": "Este servidor tiene race conditions intencionalmente para fines educativos",
		"endpoints": map[string]string{
			"health":   "/health",
			"asientos": "/asientos",
			"reservar": "/reservar",
			"liberar":  "/liberar",
			"estado":   "/estado",
			"reset":    "/reset",
		},
		"timestamp": time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// healthHandler verifica el estado del servidor
func healthHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	response := map[string]interface{}{
		"status":    "ok",
		"servidor":  servidorID,
		"timestamp": time.Now(),
		"uptime":    "activo",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// asientosHandler devuelve todos los asientos
func asientosHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	if r.Method == "OPTIONS" {
		return
	}
	
	if r.Method != "GET" {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	
	asientos := sistema.ObtenerTodosLosAsientos()
	
	response := map[string]interface{}{
		"servidor":  servidorID,
		"asientos":  asientos,
		"total":     len(asientos),
		"timestamp": time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// asientoHandler devuelve información de un asiento específico
func asientoHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	if r.Method != "GET" {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	
	// Extraer número de asiento de la URL
	numeroStr := r.URL.Path[len("/asiento/"):]
	numero, err := strconv.Atoi(numeroStr)
	if err != nil {
		http.Error(w, "Número de asiento inválido", http.StatusBadRequest)
		return
	}
	
	asiento, err := sistema.ObtenerAsiento(numero)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"servidor":  servidorID,
		"asiento":   asiento,
		"timestamp": time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ReservaRequest representa una solicitud de reserva
type ReservaRequest struct {
	Numero  int    `json:"numero"`
	Cliente string `json:"cliente"`
}

// reservarHandler maneja las reservas de asientos
func reservarHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	if r.Method == "OPTIONS" {
		return
	}
	
	if r.Method != "POST" {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	
	var req ReservaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}
	
	// Validar datos
	if req.Numero <= 0 || req.Cliente == "" {
		http.Error(w, "Número de asiento y cliente son requeridos", http.StatusBadRequest)
		return
	}
	
	// Log de la solicitud
	log.Printf("🎫 [%s] Intentando reservar asiento %d para %s", servidorID, req.Numero, req.Cliente)
	
	// AQUÍ ESTÁ EL PROBLEMA: Race condition
	err := sistema.ReservarAsiento(req.Numero, req.Cliente)
	if err != nil {
		log.Printf("❌ [%s] Error al reservar asiento %d: %s", servidorID, req.Numero, err.Error())
		
		response := map[string]interface{}{
			"success":   false,
			"error":     err.Error(),
			"servidor":  servidorID,
			"timestamp": time.Now(),
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	log.Printf("✅ [%s] Asiento %d reservado exitosamente para %s", servidorID, req.Numero, req.Cliente)
	
	// Obtener asiento actualizado
	asiento, _ := sistema.ObtenerAsiento(req.Numero)
	
	response := map[string]interface{}{
		"success":   true,
		"message":   "Asiento reservado exitosamente",
		"asiento":   asiento,
		"servidor":  servidorID,
		"timestamp": time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// LiberarRequest representa una solicitud de liberación
type LiberarRequest struct {
	Numero int `json:"numero"`
}

// liberarHandler maneja la liberación de asientos
func liberarHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	if r.Method == "OPTIONS" {
		return
	}
	
	if r.Method != "POST" {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	
	var req LiberarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}
	
	if req.Numero <= 0 {
		http.Error(w, "Número de asiento requerido", http.StatusBadRequest)
		return
	}
	
	log.Printf("🔓 [%s] Liberando asiento %d", servidorID, req.Numero)
	
	err := sistema.LiberarAsiento(req.Numero)
	if err != nil {
		log.Printf("❌ [%s] Error al liberar asiento %d: %s", servidorID, req.Numero, err.Error())
		
		response := map[string]interface{}{
			"success":   false,
			"error":     err.Error(),
			"servidor":  servidorID,
			"timestamp": time.Now(),
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	log.Printf("✅ [%s] Asiento %d liberado exitosamente", servidorID, req.Numero)
	
	response := map[string]interface{}{
		"success":   true,
		"message":   "Asiento liberado exitosamente",
		"servidor":  servidorID,
		"timestamp": time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// estadoHandler devuelve el estado del sistema
func estadoHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	if r.Method == "OPTIONS" {
		return
	}
	
	if r.Method != "GET" {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	
	estado := sistema.ObtenerEstado()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(estado)
}

// resetHandler reinicia el sistema
func resetHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	if r.Method == "OPTIONS" {
		return
	}
	
	if r.Method != "POST" {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	
	log.Printf("🔄 [%s] Reiniciando sistema...", servidorID)
	
	// Reinicializar sistema
	sistema = models.NewSistemaReservas(servidorID, 50)
	
	log.Printf("✅ [%s] Sistema reiniciado", servidorID)
	
	response := map[string]interface{}{
		"success":   true,
		"message":   "Sistema reiniciado exitosamente",
		"servidor":  servidorID,
		"timestamp": time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}