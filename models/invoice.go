package models

import "gorm.io/gorm"

type Invoice struct {
	gorm.Model
	UserID      uint    `json:"user_id"`
	FileName    string  `json:"file_name"`
	OriginalName string `json:"original_name"`
	TotalAmount float64 `json:"total_amount"`
	CO2Estimate float64 `json:"co2_estimate"`
	TextPreview string  `json:"text_preview"`
}
