package main

import "time"

// Tenant représente une entreprise cliente (multi-tenant).
type Tenant struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	Siret     string    `db:"siret"`
	Plan      string    `db:"plan"`
	CreatedAt time.Time `db:"created_at"`
}

// User représente un utilisateur rattaché à un tenant.
type User struct {
	ID           int64     `db:"id"`
	TenantID     int64     `db:"tenant_id"`
	Email        string    `db:"email"`
	Role         string    `db:"role"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
}

// Entry représente une ligne comptable ou une activité saisie par un tenant.
type Entry struct {
	ID        int64     `db:"id"`
	TenantID  int64     `db:"tenant_id"`
	Type      string    `db:"type"`
	Amount    float64   `db:"amount"`
	Currency  string    `db:"currency"`
	Date      time.Time `db:"date"`
	Category  *string   `db:"category"`
	Source    *string   `db:"source"`
	CreatedAt time.Time `db:"created_at"`
}

// Emission représente le résultat d'un calcul de CO2e lié à une entrée.
type Emission struct {
	ID                 int64     `db:"id"`
	EntryID            int64     `db:"entry_id"`
	TenantID           int64     `db:"tenant_id"`
	Scope              string    `db:"scope"` // "1","2","3"
	TCO2e              float64   `db:"tco2e"`
	MethodologyVersion string    `db:"methodology_version"`
	ComputedAt         time.Time `db:"computed_at"`
}

// Document représente un document importé (facture EDF, contrat énergie, etc.).
type Document struct {
	ID           int64     `db:"id"`
	TenantID     int64     `db:"tenant_id"`
	OriginalName string    `db:"original_name"`
	MimeType     string    `db:"mime_type"`
	SizeBytes    int64     `db:"size_bytes"`
	StoragePath  string    `db:"storage_path"`
	Source       *string   `db:"source"`
	Kind         *string   `db:"kind"`
	CreatedAt    time.Time `db:"created_at"`
}
