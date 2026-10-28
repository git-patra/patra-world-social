package model

// RNG is injected by each match. JS Math.random and Go random are not parity guarantees.
type RNG interface{ Intn(n int) int }
