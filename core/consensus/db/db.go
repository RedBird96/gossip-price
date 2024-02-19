package db

import (
	"context"
	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"gossip-price/core/global"
	"log"
)

type Database struct {
	Conn *pgxpool.Pool
}

type RateRepository interface {
	CreateRate(user *Rate) (*Rate, error)
	GetRate(id string) (*Rate, error)
}

func NewDatabase() *Database {
	// urlExample := "postgres://username:password@localhost:5432/database_name"
	dbConfig, err := pgxpool.ParseConfig(global.GPDatabaseUrl)
	if err != nil {
		return nil
	}

	// Register UUID support
	dbConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxuuid.Register(conn.TypeMap())
		return nil
	}

	conn, err := pgxpool.NewWithConfig(context.Background(), dbConfig)
	if err != nil {
		return nil
	}

	err = conn.Ping(context.Background())
	if err != nil {
		log.Print("Error connecting to the database. ")
		return nil
	}
	return &Database{Conn: conn}
}

func (d *Database) CreateRate(user *Rate) (*Rate, error) {
	sql := `
	INSERT INTO rate (id, price, first_signer, sign_data, lastsigned_time, created_time)
	VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := d.Conn.Exec(context.Background(),
		sql, user.ID, user.Price, user.First_Signer, user.Sign_Data, user.LastSigned_Time, user.Created_Time)
	if err != nil {
		return nil, err
	}
	log.Printf("Message(%s) is added to database", user.ID)
	return user, nil
}

func (d *Database) ExistCheck(msgsId string) bool {
	sql := `
	select * from rate where id = $1`
	res, err := d.Conn.Exec(context.Background(),
		sql, msgsId)
	if err != nil || res.String() == "SELECT 0" {
		return false
	}
	return true
}
