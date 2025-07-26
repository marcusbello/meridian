package repo

import (
	"context"
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

type Listing struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`       
	Category    string  `json:"category"`    // e.g., "house", "apartment", "land"
	PicturesURL string  `json:"pictures_url"` // URL to the listing's pictures
	Negotiable  bool    `json:"negotiable"`  // Whether the price is negotiable
	Type        string  `json:"type"`        // e.g., "sale", "rent"
	Location    string  `json:"location"`    // e.g., "New York", "San Francisco"
	CreatedAt   string  `json:"created_at"`  // ISO 8601 format
	UpdatedAt   string  `json:"updated_at"`  // ISO 8601 format
	Featured    bool    `json:"featured"`    // Whether the listing is featured
	UserID      int     `json:"user_id"`     // ID of the user who created the listing
}

type ListingFilter struct {
	Category   string
	PriceMin   float64
	PriceMax   float64
	Negotiable bool
	Type       string
	Location   string
	Featured   bool
	SortBy     string
	SortOrder  string
	Limit      int
	Offset     int
	UserID     string
}

type ListingRepository interface {
	AddListing(ctx context.Context, listing Listing) error
	GetListing(ctx context.Context, id int) (Listing, error)
	UpdateListing(ctx context.Context, listing Listing) error
	DeleteListing(ctx context.Context, id int) error
	ListListings(ctx context.Context, filter ListingFilter) ([]Listing, error)
}

func NewListingRepository(db *sql.DB, logger *log.Logger) ListingRepository {
	// Initialize and return a concrete implementation of Repository
	return &listingRepoImpl{
		db:     db,
		logger: logger,
	}
}

type listingRepoImpl struct {
	db     *sql.DB
	logger *log.Logger
}

// AddListing implements Repository.
func (r *listingRepoImpl) AddListing(ctx context.Context, listing Listing) error {
	// Example implementation of adding a listing to the database
	query := `INSERT INTO listings (title, description, price, category, pictures_url, negotiable, type, location, created_at, updated_at, featured, user_id) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW(), ?, ?)`
	_, err := r.db.ExecContext(ctx, query, listing.Title, listing.Description, listing.Price, listing.Category,
		listing.PicturesURL, listing.Negotiable, listing.Type, listing.Location, listing.Featured, listing.UserID)
	if err != nil {
		r.logger.Println("Error adding listing:", err)
		return err
	}
	return nil
}

// DeleteListing implements Repository.
func (r *listingRepoImpl) DeleteListing(ctx context.Context, id int) error {
	panic("unimplemented")
}

// GetListing implements Repository.
func (r *listingRepoImpl) GetListing(ctx context.Context, id int) (Listing, error) {
	var listing Listing
	query := `SELECT id, title, description, price, category, pictures_url, negotiable, type, location, created_at, updated_at, featured, user_id 
			  FROM listings WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)
	err := row.Scan(&listing.ID, &listing.Title, &listing.Description, &listing.Price, &listing.Category,
		&listing.PicturesURL, &listing.Negotiable, &listing.Type, &listing.Location, &listing.CreatedAt,
		&listing.UpdatedAt, &listing.Featured, &listing.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return Listing{}, nil // No listing found
		}
		r.logger.Println("Error getting listing:", err)
		return Listing{}, err
	}
	return listing, nil
}

// ListListings implements Repository.
func (r *listingRepoImpl) ListListings(ctx context.Context, filter ListingFilter) ([]Listing, error) {
	var listings []Listing
	query := "SELECT id, title, description, price, category, pictures_url, negotiable, type, location, created_at, updated_at, featured, user_id FROM listings WHERE 1=1"
	args := []any{}
	if filter.Category != "" {
		query += " AND category = ?"
		args = append(args, filter.Category)
	}
	if filter.PriceMin > 0 {
		query += " AND price >= ?"
		args = append(args, filter.PriceMin)
	}
	if filter.PriceMax > 0 {
		query += " AND price <= ?"
		args = append(args, filter.PriceMax)
	}
	if filter.Negotiable {
		query += " AND negotiable = ?"
		args = append(args, filter.Negotiable)
	}
	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}
	if filter.Featured {
		query += " AND featured = ?"
		args = append(args, filter.Featured)
	}
	if filter.Location != "" {
		query += " AND location = ?"
		args = append(args, filter.Location)
	}
	if filter.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, filter.UserID)
	}
	if filter.SortBy != "" {
		query += " ORDER BY " + filter.SortBy
		if filter.SortOrder == "desc" {
			query += " DESC"
		} else {
			query += " ASC"
		}
	}
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Println("Error querying listings:", err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var listing Listing
		if err := rows.Scan(&listing.ID, &listing.Title, &listing.Description, &listing.Price, &listing.Category, &listing.PicturesURL, &listing.Negotiable, &listing.Type, &listing.Location, &listing.CreatedAt, &listing.UpdatedAt, &listing.Featured, &listing.UserID); err != nil {
			r.logger.Println("Error scanning listing:", err)
			return nil, err
		}
		listings = append(listings, listing)
	}
	if err := rows.Err(); err != nil {
		r.logger.Println("Error iterating over listings:", err)
		return nil, err
	}
	return listings, nil
}

// UpdateListing implements Repository.
func (r *listingRepoImpl) UpdateListing(ctx context.Context, listing Listing) error {
	panic("unimplemented")
}
