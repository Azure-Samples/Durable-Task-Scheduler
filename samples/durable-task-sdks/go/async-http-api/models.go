package main

import "time"

type operationRequest struct {
	ProcessingTime int `json:"processing_time"`
}

type operationInput struct {
	OperationID    string `json:"operation_id"`
	ProcessingTime int    `json:"processing_time"`
}

type operationResult struct {
	OperationID string  `json:"operation_id"`
	Status      string  `json:"status"`
	Result      string  `json:"result"`
	ProcessedAt float64 `json:"processed_at"`
}

type startResponse struct {
	OperationID string `json:"operation_id"`
	StatusURL   string `json:"status_url"`
}

type statusResponse struct {
	OperationID string           `json:"operation_id"`
	Status      string           `json:"status"`
	LastUpdated time.Time        `json:"last_updated"`
	Result      *operationResult `json:"result,omitempty"`
	Error       string           `json:"error,omitempty"`
}
