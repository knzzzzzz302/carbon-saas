package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewDB initialise un pool de connexions Postgres et exécute les migrations SQL basiques.
func NewDB(cfg Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	// Test de connexion léger
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	// Migrations SQL simples (idempotentes) via le fichier sql_schema.sql.
	if err := runMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

// runMigrations exécute le contenu de sql_schema.sql (CREATE TABLE IF NOT EXISTS ...).
// C'est volontairement simple pour l'étape MVP, sans système de versionning complexe.
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	data, err := os.ReadFile("sql_schema.sql")
	if err != nil {
		// On tente aussi le chemin relatif au répertoire services/api pour plus de robustesse.
		altData, altErr := os.ReadFile("./services/api/sql_schema.sql")
		if altErr != nil {
			return err
		}
		data = altData
	}

	sql := string(data)
	if sql == "" {
		return nil
	}

	// On exécute en une seule commande ; le fichier ne contient que des CREATE TABLE IF NOT EXISTS.
	_, err = pool.Exec(ctx, sql)
	return err
}
