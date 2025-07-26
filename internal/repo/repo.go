package repo

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)


type Repository interface {
	ListingRepository
	PurchaseRepository
}

type repositoryImpl struct {
	ListingRepository
	PurchaseRepository
}

func NewRepository(db *sql.DB, logger *log.Logger) Repository {
	return &repositoryImpl{
		ListingRepository:  NewListingRepository(db, logger),
		PurchaseRepository: NewPurchaseRepository(db, logger),
	}
}



