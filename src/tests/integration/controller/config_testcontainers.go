package controller_test

import (
	"context"
	"fmt"
	"log"

	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(ctx context.Context) (*gorm.DB, func(), error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "help_dev_test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"), 
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, nil, err
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, nil, err
	}

	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=help_dev_test sslmode=disable",
		host, port.Port())

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		container.Terminate(ctx)
		return nil, nil, err
	}

	cleanup := func() {
		if err := container.Terminate(ctx); err != nil {
			log.Printf("Erro ao parar o container: %v", err)
		}
	}
	db.AutoMigrate(&entity.UserEntity{})

	return db, cleanup, nil
}