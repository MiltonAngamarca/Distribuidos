package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Asiento representa un asiento en la base de datos
type Asiento struct {
	Numero     int       `bson:"numero" json:"numero"`
	Disponible bool      `bson:"disponible" json:"disponible"`
	Cliente    string    `bson:"cliente,omitempty" json:"cliente,omitempty"`
	ServerID   string    `bson:"server_id" json:"server_id"`
	UpdatedAt  time.Time `bson:"updated_at" json:"updated_at"`
}

// Server es la estructura principal de nuestro servidor de reservas
type Server struct {
	node       *Node
	collection *mongo.Collection
	serverID   string
}

// NewServer crea una nueva instancia del servidor
func NewServer(node *Node, collection *mongo.Collection, serverID string) *Server {
	return &Server{
		node:       node,
		collection: collection,
		serverID:   serverID,
	}
}

// --- HTTP Handlers ---

// handleGetAsientos devuelve el estado de todos los asientos desde la BD
func (s *Server) handleGetAsientos(w http.ResponseWriter, r *http.Request) {
	// Configurar headers CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	cursor, err := s.collection.Find(context.Background(), bson.M{})
	if err != nil {
		http.Error(w, "Failed to fetch seats", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var asientos []Asiento
	if err = cursor.All(context.Background(), &asientos); err != nil {
		http.Error(w, "Failed to decode seats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"asientos":  asientos,
		"server_id": s.serverID,
	})
}

// handleReservarAsiento gestiona la reserva de un asiento usando Ricart-Agrawala
func (s *Server) handleReservarAsiento(w http.ResponseWriter, r *http.Request) {
	// Configurar headers CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	var req struct {
		Numero  int    `json:"numero"`
		Cliente string `json:"cliente"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 1. Solicitar acceso a la sección crítica
	log.Printf("[%s] Requesting CS to reserve seat %d", s.serverID, req.Numero)
	s.node.RequestCS()
	log.Printf("[%s] Granted CS to reserve seat %d", s.serverID, req.Numero)

	// Defer la liberación de la sección crítica
	defer s.node.ReleaseCS()

	// 2. Una vez dentro de la sección crítica, realizar la operación
	var asiento Asiento
	err := s.collection.FindOne(context.Background(), bson.M{"numero": req.Numero}).Decode(&asiento)
	if err != nil {
		http.Error(w, "Asiento no encontrado", http.StatusNotFound)
		return
	}

	if !asiento.Disponible {
		response := map[string]interface{}{
			"success": false,
			"message": "Asiento ya está ocupado",
			"server_id": s.serverID,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Actualizar el asiento
	update := bson.M{
		"$set": bson.M{
			"disponible": false,
			"cliente":    req.Cliente,
			"server_id":  s.serverID,
			"updated_at": time.Now(),
		},
	}

	_, err = s.collection.UpdateOne(context.Background(), bson.M{"numero": req.Numero}, update)
	if err != nil {
		http.Error(w, "Failed to update seat", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Asiento reservado exitosamente",
		"server_id": s.serverID,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleLiberarAsiento gestiona la liberación de un asiento usando Ricart-Agrawala
func (s *Server) handleLiberarAsiento(w http.ResponseWriter, r *http.Request) {
	// Configurar headers CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	var req struct {
		Numero int `json:"numero"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Solicitar acceso a la sección crítica
	s.node.RequestCS()
	defer s.node.ReleaseCS()

	// Verificar que el asiento existe y está ocupado
	var asiento Asiento
	err := s.collection.FindOne(context.Background(), bson.M{"numero": req.Numero}).Decode(&asiento)
	if err != nil {
		http.Error(w, "Seat not found", http.StatusNotFound)
		return
	}

	if asiento.Disponible {
		http.Error(w, "Seat is already available", http.StatusBadRequest)
		return
	}

	// Liberar el asiento
	update := bson.M{
		"$set": bson.M{
			"disponible": true,
			"cliente":    "",
			"server_id":  s.serverID,
			"updated_at": time.Now(),
		},
	}

	_, err = s.collection.UpdateOne(context.Background(), bson.M{"numero": req.Numero}, update)
	if err != nil {
		http.Error(w, "Failed to update seat", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Asiento liberado exitosamente",
		"server_id": s.serverID,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleInternalMessage es el endpoint para la comunicación entre nodos
func (s *Server) handleInternalMessage(w http.ResponseWriter, r *http.Request) {
	var msg Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid message", http.StatusBadRequest)
		return
	}

	// Procesar el mensaje en una goroutine para no bloquear
	go s.node.handleMessage(msg)

	w.WriteHeader(http.StatusOK)
}

// handleHealthCheck comprueba la salud del servidor
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"server_id": s.serverID,
		"time":      s.node.Clock.GetTime(),
	})
}

// --- Main y Setup ---

func main() {
	// 1. Leer configuración del entorno
	serverID := os.Getenv("SERVER_ID")
	if serverID == "" {
		log.Fatal("SERVER_ID must be set")
	}

	peersStr := os.Getenv("PEERS") // e.g., "server1:8081,server2:8082"
	if peersStr == "" {
		log.Fatal("PEERS must be set")
	}
	
	// Parse peers - they come as "server1,server2,server3" but we need full URLs
	rawPeers := strings.Split(peersStr, ",")
	var peers []string
	
	// Convert peer names to proper URLs for Docker networking
	for _, peer := range rawPeers {
		if peer != serverID { // Don't include self
			switch peer {
			case "server1":
				peers = append(peers, "server1")
			case "server2":
				peers = append(peers, "server2")
			case "server3":
				peers = append(peers, "server3")
			default:
				peers = append(peers, peer)
			}
		}
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://mongo:27017"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("[%s] Starting with peers: %v", serverID, peers)

	// 2. Conectar a MongoDB
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	collection := client.Database("reservations_db_distributed").Collection("seats")

	// 3. Inicializar el nodo de Ricart-Agrawala
	node := NewNode(serverID, peers)

	// 4. Crear el servidor
	server := NewServer(node, collection, serverID)

	// 5. Inicializar asientos si es necesario (solo lo hace un nodo)
	if serverID == rawPeers[0] { // El primer peer es el encargado
		initializeSeats(collection)
	}

	// 6. Configurar rutas
	r := mux.NewRouter()
	
	// Middleware CORS para manejar preflight requests
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	})
	
	// Endpoints públicos
	r.HandleFunc("/asientos", server.handleGetAsientos).Methods("GET")
	r.HandleFunc("/reservar", server.handleReservarAsiento).Methods("POST", "OPTIONS")
	r.HandleFunc("/liberar", server.handleLiberarAsiento).Methods("POST", "OPTIONS")
	r.HandleFunc("/health", server.handleHealthCheck).Methods("GET")

	// Endpoint interno para el algoritmo
	r.HandleFunc("/internal/message", server.handleInternalMessage).Methods("POST")

	// 7. Iniciar servidor
	log.Printf("Distributed Reservation Server %s starting on port %s", serverID, port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// initializeSeats crea los asientos en la BD si no existen
func initializeSeats(collection *mongo.Collection) {
	count, err := collection.CountDocuments(context.Background(), bson.M{})
	if err != nil {
		log.Printf("Failed to count seats: %v", err)
		return
	}

	if count == 0 {
		log.Println("Initializing 20 seats in the database...")
		var asientos []interface{}
		for i := 1; i <= 20; i++ {
			asientos = append(asientos, Asiento{
				Numero:     i,
				Disponible: true,
				UpdatedAt:  time.Now(),
			})
		}
		_, err := collection.InsertMany(context.Background(), asientos)
		if err != nil {
			log.Printf("Failed to initialize seats: %v", err)
		}
	}
}