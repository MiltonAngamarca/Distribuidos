package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// LockRequest representa una solicitud de bloqueo
type LockRequest struct {
	Resource string `json:"resource"`
	ClientID string `json:"client_id"`
	TTL      int    `json:"ttl"` // Time to live en segundos
}

// LockResponse representa la respuesta de un bloqueo
type LockResponse struct {
	Success   bool   `json:"success"`
	LockID    string `json:"lock_id,omitempty"`
	Message   string `json:"message,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}

// Lock representa un bloqueo activo
type Lock struct {
	ID        string    `bson:"_id" json:"id"`
	Resource  string    `bson:"resource" json:"resource"`
	ClientID  string    `bson:"client_id" json:"client_id"`
	ExpiresAt time.Time `bson:"expires_at" json:"expires_at"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

// LockCoordinator maneja los bloqueos distribuidos
type LockCoordinator struct {
	locks      map[string]*Lock
	mutex      sync.RWMutex
	collection *mongo.Collection
}

// NewLockCoordinator crea un nuevo coordinador de bloqueos
func NewLockCoordinator(collection *mongo.Collection) *LockCoordinator {
	lc := &LockCoordinator{
		locks:      make(map[string]*Lock),
		collection: collection,
	}
	
	// Iniciar limpieza periódica de bloqueos expirados
	go lc.cleanupExpiredLocks()
	
	return lc
}

// AcquireLock intenta adquirir un bloqueo
func (lc *LockCoordinator) AcquireLock(resource, clientID string, ttl int) (*LockResponse, error) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	// Verificar si ya existe un bloqueo activo para este recurso
	if existingLock, exists := lc.locks[resource]; exists {
		if time.Now().Before(existingLock.ExpiresAt) {
			return &LockResponse{
				Success: false,
				Message: fmt.Sprintf("Resource %s is already locked by client %s", resource, existingLock.ClientID),
			}, nil
		}
		// El bloqueo ha expirado, eliminarlo
		delete(lc.locks, resource)
		lc.collection.DeleteOne(context.Background(), bson.M{"_id": existingLock.ID})
	}

	// Crear nuevo bloqueo
	lockID := fmt.Sprintf("%s_%s_%d", resource, clientID, time.Now().UnixNano())
	expiresAt := time.Now().Add(time.Duration(ttl) * time.Second)
	
	lock := &Lock{
		ID:        lockID,
		Resource:  resource,
		ClientID:  clientID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	// Guardar en memoria y MongoDB
	lc.locks[resource] = lock
	_, err := lc.collection.InsertOne(context.Background(), lock)
	if err != nil {
		delete(lc.locks, resource)
		return nil, fmt.Errorf("failed to save lock to database: %v", err)
	}

	return &LockResponse{
		Success:   true,
		LockID:    lockID,
		Message:   "Lock acquired successfully",
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

// ReleaseLock libera un bloqueo
func (lc *LockCoordinator) ReleaseLock(resource, clientID string) (*LockResponse, error) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	lock, exists := lc.locks[resource]
	if !exists {
		return &LockResponse{
			Success: false,
			Message: "No lock found for this resource",
		}, nil
	}

	if lock.ClientID != clientID {
		return &LockResponse{
			Success: false,
			Message: "Lock belongs to a different client",
		}, nil
	}

	// Eliminar de memoria y MongoDB
	delete(lc.locks, resource)
	_, err := lc.collection.DeleteOne(context.Background(), bson.M{"_id": lock.ID})
	if err != nil {
		log.Printf("Failed to delete lock from database: %v", err)
	}

	return &LockResponse{
		Success: true,
		Message: "Lock released successfully",
	}, nil
}

// GetLockStatus obtiene el estado de un bloqueo
func (lc *LockCoordinator) GetLockStatus(resource string) (*Lock, bool) {
	lc.mutex.RLock()
	defer lc.mutex.RUnlock()

	lock, exists := lc.locks[resource]
	if !exists {
		return nil, false
	}

	if time.Now().After(lock.ExpiresAt) {
		// El bloqueo ha expirado
		go func() {
			lc.mutex.Lock()
			delete(lc.locks, resource)
			lc.collection.DeleteOne(context.Background(), bson.M{"_id": lock.ID})
			lc.mutex.Unlock()
		}()
		return nil, false
	}

	return lock, true
}

// cleanupExpiredLocks limpia periódicamente los bloqueos expirados
func (lc *LockCoordinator) cleanupExpiredLocks() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		lc.mutex.Lock()
		now := time.Now()
		
		for resource, lock := range lc.locks {
			if now.After(lock.ExpiresAt) {
				delete(lc.locks, resource)
				lc.collection.DeleteOne(context.Background(), bson.M{"_id": lock.ID})
				log.Printf("Cleaned up expired lock for resource: %s", resource)
			}
		}
		lc.mutex.Unlock()
	}
}

// HTTP Handlers

func (lc *LockCoordinator) handleAcquireLock(w http.ResponseWriter, r *http.Request) {
	var req LockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.TTL <= 0 {
		req.TTL = 300 // Default 5 minutes
	}

	response, err := lc.AcquireLock(req.Resource, req.ClientID, req.TTL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (lc *LockCoordinator) handleReleaseLock(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Resource string `json:"resource"`
		ClientID string `json:"client_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response, err := lc.ReleaseLock(req.Resource, req.ClientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (lc *LockCoordinator) handleGetLockStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resource := vars["resource"]

	lock, exists := lc.GetLockStatus(resource)
	
	response := map[string]interface{}{
		"resource": resource,
		"locked":   exists,
	}
	
	if exists {
		response["lock"] = lock
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (lc *LockCoordinator) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func main() {
	// Conectar a MongoDB
	mongoURI := "mongodb://mongo:27017"
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer client.Disconnect(context.Background())

	// Verificar conexión
	if err := client.Ping(context.Background(), nil); err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}

	collection := client.Database("locks_db").Collection("locks")
	
	// Crear coordinador de bloqueos
	coordinator := NewLockCoordinator(collection)

	// Configurar rutas
	r := mux.NewRouter()

       // ...existing code...

	r.HandleFunc("/acquire", coordinator.handleAcquireLock).Methods("POST", "OPTIONS")
	r.HandleFunc("/release", coordinator.handleReleaseLock).Methods("POST", "OPTIONS")
	r.HandleFunc("/status/{resource}", coordinator.handleGetLockStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/health", coordinator.handleHealthCheck).Methods("GET", "OPTIONS")


	port := ":8080"
	log.Printf("Lock Coordinator starting on port %s", port)
	log.Fatal(http.ListenAndServe(port, r))
}