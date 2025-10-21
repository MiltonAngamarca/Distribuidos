package main

import "sync"

// LamportClock implementa un reloj lógico de Lamport.
// Es seguro para su uso concurrente.
type LamportClock struct {
	time int64
	mu   sync.Mutex
}

// NewLamportClock crea una nueva instancia de LamportClock.
func NewLamportClock() *LamportClock {
	return &LamportClock{time: 0}
}

// Increment incrementa el reloj y devuelve el nuevo valor.
// Se usa antes de que ocurra un evento local.
func (c *LamportClock) Increment() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.time++
	return c.time
}

// GetTime devuelve el valor actual del reloj.
func (c *LamportClock) GetTime() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.time
}

// Witness actualiza el reloj del proceso al recibir un timestamp de otro proceso.
// Esta es la segunda regla del algoritmo de Lamport.
func (c *LamportClock) Witness(receivedTime int64) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if receivedTime > c.time {
		c.time = receivedTime
	}
	c.time++
	return c.time
}