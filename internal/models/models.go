package models

type CalculationRequest struct {
	EpsilonStart float64 `json:"epsilon_start"`
	EpsilonMin   float64 `json:"epsilon_min"`
	NStart       int     `json:"n_start"`
	NMax         int     `json:"n_max"`
	Delta        float64 `json:"delta"`
	MeshType     string  `json:"mesh_type"`
}
