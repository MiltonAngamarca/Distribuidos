package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Asiento representa un asiento en el sistema
type Asiento struct {
	Numero     int    `bson:"numero" json:"numero"`
	Disponible bool   `bson:"disponible" json:"disponible"`
	Cliente    string `bson:"cliente,omitempty" json:"cliente,omitempty"`
	ServerID   string `bson:"server_id" json:"server_id"`
	UpdatedAt  time.Time `bson:"updated_at" json:"updated_at"`
}

// LockRequest para comunicarse con el coordinador
type LockRequest struct {
	Resource string `json:"resource"`
	ClientID string `json:"client_id"`
	TTL      int    `json:"ttl"`
}

// LockResponse del coordinador
type LockResponse struct {
	Success   bool   `json:"success"`
	LockID    string `json:"lock_id,omitempty"`
	Message   string `json:"message,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}

// ReservationServer maneja las reservas de asientos
type ReservationServer struct {
	serverID         string
	coordinatorURL   string
	collection       *mongo.Collection
	asientos         map[int]*Asiento
	mutex            sync.RWMutex
	activeLocks      map[string]string // resource -> lockID
	locksMutex       sync.RWMutex
}

// NewReservationServer crea un nuevo servidor de reservas
func NewReservationServer(serverID, coordinatorURL string, collection *mongo.Collection) *ReservationServer {
	rs := &ReservationServer{
		serverID:       serverID,
		coordinatorURL: coordinatorURL,
		collection:     collection,
		asientos:       make(map[int]*Asiento),
		activeLocks:    make(map[string]string),
	}
	
	// Inicializar asientos
	rs.initializeSeats()
	
	return rs
}

// initializeSeats inicializa los asientos en la base de datos
func (rs *ReservationServer) initializeSeats() {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	// Cargar asientos existentes de la base de datos
	cursor, err := rs.collection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Printf("Error loading seats from database: %v", err)
	} else {
		for cursor.Next(context.Background()) {
			var asiento Asiento
			if err := cursor.Decode(&asiento); err == nil {
				rs.asientos[asiento.Numero] = &asiento
			}
		}
		cursor.Close(context.Background())
	}

	// Si no hay asientos, crear 20 asientos por defecto
	if len(rs.asientos) == 0 {
		for i := 1; i <= 20; i++ {
			asiento := &Asiento{
				Numero:     i,
				Disponible: true,
				ServerID:   rs.serverID,
				UpdatedAt:  time.Now(),
			}
			rs.asientos[i] = asiento
			
			// Guardar en base de datos
			_, err := rs.collection.ReplaceOne(
				context.Background(),
				bson.M{"numero": i},
				asiento,
				options.Replace().SetUpsert(true),
			)
			if err != nil {
				log.Printf("Error saving seat %d: %v", i, err)
			}
		}
		log.Printf("Initialized %d seats for server %s", len(rs.asientos), rs.serverID)
	}
}

// acquireLock solicita un bloqueo al coordinador
func (rs *ReservationServer) acquireLock(resource string, ttl int) (*LockResponse, error) {
	lockReq := LockRequest{
		Resource: resource,
		ClientID: rs.serverID,
		TTL:      ttl,
	}

	jsonData, err := json.Marshal(lockReq)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(rs.coordinatorURL+"/acquire", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var lockResp LockResponse
	if err := json.NewDecoder(resp.Body).Decode(&lockResp); err != nil {
		return nil, err
	}

	return &lockResp, nil
}

// releaseLock libera un bloqueo en el coordinador
func (rs *ReservationServer) releaseLock(resource string) error {
	releaseReq := map[string]string{
		"resource":  resource,
		"client_id": rs.serverID,
	}

	jsonData, err := json.Marshal(releaseReq)
	if err != nil {
		return err
	}

	resp, err := http.Post(rs.coordinatorURL+"/release", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ReservarAsiento reserva un asiento específico
func (rs *ReservationServer) ReservarAsiento(numero int, cliente string) (bool, string) {
	resource := fmt.Sprintf("seat_%d", numero)
	
	// Intentar adquirir bloqueo
	lockResp, err := rs.acquireLock(resource, 30) // 30 segundos TTL
	if err != nil {
		return false, fmt.Sprintf("Error acquiring lock: %v", err)
	}
	
	if !lockResp.Success {
		return false, lockResp.Message
	}

	// Guardar el lockID para liberarlo después
	rs.locksMutex.Lock()
	rs.activeLocks[resource] = lockResp.LockID
	rs.locksMutex.Unlock()

	defer func() {
		// Liberar el bloqueo al finalizar
		rs.releaseLock(resource)
		rs.locksMutex.Lock()
		delete(rs.activeLocks, resource)
		rs.locksMutex.Unlock()
	}()

	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	// Verificar si el asiento existe y está disponible
	asiento, exists := rs.asientos[numero]
	if !exists {
		return false, "Asiento no existe"
	}

	if !asiento.Disponible {
		return false, "Asiento ya está ocupado"
	}

	// Reservar el asiento
	asiento.Disponible = false
	asiento.Cliente = cliente
	asiento.UpdatedAt = time.Now()

	// Actualizar en base de datos
	_, err = rs.collection.ReplaceOne(
		context.Background(),
		bson.M{"numero": numero},
		asiento,
		options.Replace().SetUpsert(true),
	)
	if err != nil {
		// Revertir cambios en caso de error
		asiento.Disponible = true
		asiento.Cliente = ""
		return false, fmt.Sprintf("Error updating database: %v", err)
	}

	log.Printf("Server %s: Seat %d reserved by %s", rs.serverID, numero, cliente)
	return true, "Asiento reservado exitosamente"
}

// LiberarAsiento libera un asiento específico
func (rs *ReservationServer) LiberarAsiento(numero int) (bool, string) {
	resource := fmt.Sprintf("seat_%d", numero)
	
	// Intentar adquirir bloqueo
	lockResp, err := rs.acquireLock(resource, 30)
	if err != nil {
		return false, fmt.Sprintf("Error acquiring lock: %v", err)
	}
	
	if !lockResp.Success {
		return false, lockResp.Message
	}

	defer func() {
		rs.releaseLock(resource)
		rs.locksMutex.Lock()
		delete(rs.activeLocks, resource)
		rs.locksMutex.Unlock()
	}()

	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	asiento, exists := rs.asientos[numero]
	if !exists {
		return false, "Asiento no existe"
	}

	if asiento.Disponible {
		return false, "Asiento ya está disponible"
	}

	// Liberar el asiento
	asiento.Disponible = true
	asiento.Cliente = ""
	asiento.UpdatedAt = time.Now()

	// Actualizar en base de datos
	_, err = rs.collection.ReplaceOne(
		context.Background(),
		bson.M{"numero": numero},
		asiento,
		options.Replace().SetUpsert(true),
	)
	if err != nil {
		// Revertir cambios en caso de error
		asiento.Disponible = false
		return false, fmt.Sprintf("Error updating database: %v", err)
	}

	log.Printf("Server %s: Seat %d freed", rs.serverID, numero)
	return true, "Asiento liberado exitosamente"
}

// GetAsientos obtiene todos los asientos, actualizando la caché desde la base de datos
func (rs *ReservationServer) GetAsientos() (map[int]*Asiento, error) {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	// Consultar todos los asientos de la base de datos
	cursor, err := rs.collection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Printf("Error fetching seats from database: %v", err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	// Crear un nuevo mapa para la caché actualizada
	newAsientos := make(map[int]*Asiento)
	for cursor.Next(context.Background()) {
		var asiento Asiento
		if err := cursor.Decode(&asiento); err == nil {
			newAsientos[asiento.Numero] = &asiento
		}
	}

	// Reemplazar la caché antigua con la nueva
	rs.asientos = newAsientos
	log.Printf("Server %s: Cache updated with %d seats from database", rs.serverID, len(rs.asientos))

	return rs.asientos, nil
}

// HTTP Handlers

func (rs *ReservationServer) handleGetAsientos(w http.ResponseWriter, r *http.Request) {
	asientos, err := rs.GetAsientos()
	if err != nil {
		http.Error(w, "Failed to get seats", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"asientos": asientos,
		"server_id": rs.serverID,
	})
}

func (rs *ReservationServer) handleReservarAsiento(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Numero  int    `json:"numero"`
		Cliente string `json:"cliente"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Cliente == "" {
		http.Error(w, "Cliente is required", http.StatusBadRequest)
		return
	}

	success, message := rs.ReservarAsiento(req.Numero, req.Cliente)
	
	response := map[string]interface{}{
		"success": success,
		"message": message,
		"server_id": rs.serverID,
	}

	w.Header().Set("Content-Type", "application/json")
	if success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusConflict)
	}
	json.NewEncoder(w).Encode(response)
}

func (rs *ReservationServer) handleLiberarAsiento(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Numero int `json:"numero"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	success, message := rs.LiberarAsiento(req.Numero)
	
	response := map[string]interface{}{
		"success": success,
		"message": message,
		"server_id": rs.serverID,
	}

	w.Header().Set("Content-Type", "application/json")
	if success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusConflict)
	}
	json.NewEncoder(w).Encode(response)
}

func (rs *ReservationServer) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"server_id": rs.serverID,
		"time": time.Now().Format(time.RFC3339),
		"seats_count": len(rs.asientos),
	})
}

func main() {
	// Obtener configuración del entorno
	serverID := os.Getenv("SERVER_ID")
	if serverID == "" {
		serverID = "server-1"
	}

	coordinatorURL := os.Getenv("COORDINATOR_URL")
	if coordinatorURL == "" {
		coordinatorURL = "http://coordinator:8080"
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://mongo:27017"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	// Conectar a MongoDB
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer client.Disconnect(context.Background())

	// Verificar conexión
	if err := client.Ping(context.Background(), nil); err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}

	collection := client.Database("reservations_db").Collection("seats")

	// Crear servidor de reservas
	server := NewReservationServer(serverID, coordinatorURL, collection)

	// Configurar rutas
	r := mux.NewRouter()

       // ...existing code...

	r.HandleFunc("/asientos", server.handleGetAsientos).Methods("GET")
	r.HandleFunc("/reservar", server.handleReservarAsiento).Methods("POST")
	r.HandleFunc("/liberar", server.handleLiberarAsiento).Methods("POST")
	r.HandleFunc("/health", server.handleHealthCheck).Methods("GET")



	log.Printf("Reservation Server %s starting on port %s", serverID, port)
	log.Printf("Coordinator URL: %s", coordinatorURL)
	log.Fatal(http.ListenAndServe(":"+port, r))
}