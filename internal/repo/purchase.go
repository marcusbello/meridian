package repo

import (
	"context"
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

type Purchase struct {
	ID           int               `json:"id"`
	ItemID       int               `json:"item_id"`       // ID of the purchased item
	BuyerEmail   string            `json:"buyer_email"`   // Email of the buyer
	BuyerName    string            `json:"buyer_name"`    // Name of the buyer
	BuyerPhone   string            `json:"buyer_phone"`   // Phone number of the buyer
	BuyerAddress string            `json:"buyer_address"` // Address of the buyer
	CreatedAt    string            `json:"created_at"`    // ISO 8601 format
	UpdatedAt    string            `json:"updated_at"`    // ISO 8601 format
	DocumentURL  string            `json:"document_url"`  // URL to the purchase document (e.g., receipt)
	Uploads      []string          `json:"uploads"`       // URLs to any uploaded files related to the purchase
	Metadata     map[string]string `json:"metadata"`      // Additional metadata about the purchase
}

type PurchaseRepository interface {
	PurchaseItem(ctx context.Context, purchase Purchase) error
	GetUserSales(ctx context.Context, userID int) ([]Listing, error)
	GetPurchase(ctx context.Context, itemID int) (Purchase, error)
}

type purchaseRepoImpl struct {
	db     *sql.DB
	logger *log.Logger
}

// GetPurchase implements PurchaseRepository.
func (p *purchaseRepoImpl) GetPurchase(ctx context.Context, itemID int) (Purchase, error) {
	panic("unimplemented")
}

// GetUserSales implements PurchaseRepository.
func (p *purchaseRepoImpl) GetUserSales(ctx context.Context, userID int) ([]Listing, error) {
	panic("unimplemented")
}

// PurchaseItem implements PurchaseRepository.
func (p *purchaseRepoImpl) PurchaseItem(ctx context.Context, purchase Purchase) error {
	p.logger.Printf("purchase : %v", purchase)
	return nil
}

func NewPurchaseRepository(db *sql.DB, logger *log.Logger) PurchaseRepository {
	// Initialize and return a concrete implementation of Repository
	return &purchaseRepoImpl{
		db:     db,
		logger: logger,
	}
}
