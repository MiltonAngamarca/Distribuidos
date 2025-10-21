package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// Estado del nodo respecto a la sección crítica
type NodeState int

const (
	Released NodeState = iota // El nodo no está interesado en la sección crítica
	Wanted                    // El nodo quiere entrar a la sección crítica
	Held                      // El nodo está en la sección crítica
)

func (s NodeState) String() string {
	switch s {
	case Released:
		return "Released"
	case Wanted:
		return "Wanted"
	case Held:
		return "Held"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

// Mensaje intercambiado entre nodos
type Message struct {
	Type      string `json:"type"`       // "REQUEST" o "REPLY"
	Timestamp int64  `json:"timestamp"`
	NodeID    string `json:"node_id"`
}

// Node representa un proceso en el algoritmo de Ricart-Agrawala
type Node struct {
	ID    string
	Peers []string // Lista de URLs de otros nodos
	Clock *LamportClock

	State           NodeState
	RequestTime     int64
	RepliesNeeded   map[string]bool
	DeferredReplies []string

	mu sync.Mutex

	// Canal para notificar cuando se obtiene el acceso a la CS
	csGranted chan bool
}

// NewNode crea un nuevo nodo para el algoritmo
func NewNode(id string, peers []string) *Node {
	// Filter out self from peers to avoid self-messages
	var filteredPeers []string
	for _, peer := range peers {
		if peer != id {
			filteredPeers = append(filteredPeers, peer)
		}
	}
	
	n := &Node{
		ID:              id,
		Peers:           filteredPeers,
		Clock:           NewLamportClock(),
		State:           Released,
		RepliesNeeded:   make(map[string]bool),
		DeferredReplies: []string{},
		csGranted:       make(chan bool, 1),
	}
	return n
}

// RequestCS intenta obtener acceso a la sección crítica
func (n *Node) RequestCS() {
	n.mu.Lock()
	n.State = Wanted
	n.RequestTime = n.Clock.Increment()

	// Necesitamos respuesta de todos los peers
	for _, peer := range n.Peers {
		if peer != n.ID {
			n.RepliesNeeded[peer] = true
		}
	}
	n.mu.Unlock()

	// Si no hay otros peers, entramos directamente
	if len(n.Peers) <= 1 {
		n.enterCS()
		return
	}

	// Enviar REQUEST a todos los demás nodos
	msg := Message{
		Type:      "REQUEST",
		Timestamp: n.RequestTime,
		NodeID:    n.ID,
	}
	n.broadcast(msg)

	// Esperar a que se conceda el acceso
	<-n.csGranted
}

// ReleaseCS libera la sección crítica
// ReleaseCS libera la sección crítica
func (n *Node) ReleaseCS() {
	n.mu.Lock()
	n.State = Released
	
	log.Printf("[%s] Releasing critical section, sending %d deferred replies", 
		n.ID, len(n.DeferredReplies))
	
	// Enviar todos los replies que habíamos pospuesto
	for _, nodeID := range n.DeferredReplies {
		log.Printf("[%s] Sending deferred reply to %s", n.ID, nodeID)
		n.sendReply(nodeID)
	}
	n.DeferredReplies = []string{}
	n.mu.Unlock()

	log.Printf("[%s] Released critical section", n.ID)
}

// enterCS es llamado cuando el nodo obtiene acceso a la CS
func (n *Node) enterCS() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.State == Wanted {
		log.Printf("[%s] Entering critical section", n.ID)
		n.State = Held
		n.csGranted <- true
	}
}

// handleMessage procesa los mensajes entrantes (REQUEST/REPLY)
func (n *Node) handleMessage(msg Message) {
	// Actualizar el reloj de Lamport al recibir cualquier mensaje
	n.Clock.Witness(msg.Timestamp)

	log.Printf("[%s] Received %s message from %s (timestamp: %d)", 
		n.ID, msg.Type, msg.NodeID, msg.Timestamp)

	switch msg.Type {
	case "REQUEST":
		n.handleRequest(msg)
	case "REPLY":
		n.handleReply(msg)
	}
}

// handleRequest gestiona una petición de acceso a la CS
func (n *Node) handleRequest(msg Message) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Actualizar el reloj de Lamport con el timestamp del mensaje
	n.Clock.Witness(msg.Timestamp)

	// La decisión de responder se basa en el estado y el timestamp
	shouldReply := n.State == Released ||
		(n.State == Wanted && (msg.Timestamp < n.RequestTime || 
			(msg.Timestamp == n.RequestTime && msg.NodeID < n.ID)))

	log.Printf("[%s] Received REQUEST from %s (ts:%d vs my:%d, state:%s)", 
		n.ID, msg.NodeID, msg.Timestamp, n.RequestTime, n.State)

	if shouldReply {
		log.Printf("[%s] Sending reply to %s", n.ID, msg.NodeID)
		n.sendReply(msg.NodeID)
	} else {
		// Posponer la respuesta - usar NodeID directamente
		log.Printf("[%s] Deferring reply to %s (reason: state=%s, ts_cmp=%t, id_cmp=%t)",
			n.ID, msg.NodeID, n.State, msg.Timestamp < n.RequestTime, msg.NodeID < n.ID)
		n.DeferredReplies = append(n.DeferredReplies, msg.NodeID)
	}
}

// handleReply gestiona una respuesta a nuestra petición
func (n *Node) handleReply(msg Message) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.State == Wanted {
		// Usar el NodeID del mensaje para eliminar de RepliesNeeded
		delete(n.RepliesNeeded, msg.NodeID)
		log.Printf("[%s] Got reply from %s. Needed: %d", n.ID, msg.NodeID, len(n.RepliesNeeded))

		// Si ya tenemos todas las respuestas, podemos entrar a la CS
		if len(n.RepliesNeeded) == 0 {
			n.enterCS()
		}
	}
}

// broadcast envía un mensaje a todos los peers
func (n *Node) broadcast(msg Message) {
	for _, peerURL := range n.Peers {
		if peerURL != n.ID { // No nos enviamos a nosotros mismos
			go n.sendMessage(peerURL, msg)
		}
	}
}

// sendReply envía una respuesta a un nodo específico
func (n *Node) sendReply(peerID string) {
	reply := Message{
		Type:      "REPLY",
		Timestamp: n.Clock.Increment(),
		NodeID:    n.ID,
	}
	go n.sendMessage(peerID, reply)
	log.Printf("[%s] Sent reply to %s", n.ID, peerID)
}

// sendMessage envía un mensaje a un peer
func (n *Node) sendMessage(peerID string, msg Message) {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[%s] Error marshalling message: %v", n.ID, err)
		return
	}

	// No enviamos mensajes a nosotros mismos
	if peerID == n.ID {
		return
	}
	
	// Obtener la URL del peer usando la función findPeerURL
	url := n.findPeerURL(peerID)

	_, err = http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[%s] Failed to send message to %s at %s: %v", n.ID, peerID, url, err)
	}
}

// findPeerURL encuentra la URL de un peer por su ID
func (n *Node) findPeerURL(nodeID string) string {
	// Mapear IDs de nodos a URLs de servicios Docker
	switch nodeID {
	case "server1":
		return "http://server1:8081/internal/message"
	case "server2":
		return "http://server2:8082/internal/message"
	case "server3":
		return "http://server3:8083/internal/message"
	default:
		// Fallback para otros casos
		return fmt.Sprintf("http://%s/internal/message", nodeID)
	}
}