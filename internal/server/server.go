package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"github.com/marcusbello/meridian/internal/repo"
)


var jwtSecret = []byte("super-secret-key")

type contextKey string

const userCtxKey = contextKey("user")



type Server struct {
	Logger    *log.Logger
	DB        *sql.DB
	templates *template.Template
	repo repo.Repository
}

func NewServer() *Server {
	logger := log.Default()

	dsn := "root:1234@tcp(127.0.0.1:3306)/meridiandb?parseTime=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		logger.Fatalf("DB connection failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.Fatalf("DB unreachable: %v", err)
	}

	// Define template functions
	funcs := template.FuncMap{
		"toJson": toJson,
	}

	// Parse all templates in templates/ folder
	tmpl := template.Must(template.New("").Funcs(funcs).ParseGlob("templates/*.html"))

	// Initialize the repository with the database connection and logger
	repository := repo.NewRepository(db, logger)

	return &Server{
		Logger:    logger,
		DB:        db,
		templates: tmpl,
		repo: 	repository,
	}
}

func (s *Server) render(w http.ResponseWriter, view string, data any) {
	err := s.templates.ExecuteTemplate(w, view, data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := ctx.Value(userCtxKey).(string)

	s.Logger.Println("Dashboard accessed by:", user)
	// fetch user-specific data here, e.g., listings
	listings, err := s.repo.ListListings(ctx, repo.ListingFilter{
		// UserID: user, // Assuming UserID is stored in the context
		Limit: 10,
		Offset: 0,
		SortBy: "created_at",
	})
	if err != nil {
		s.Logger.Println("Error fetching listings:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	data := map[string]any{
		"Title":    "Dashboard",
		"Username": user,
		"Data": map[string]any{
			"Message": "Welcome to your dashboard!",
			"Listings": listings,
		},
	}
	// Render the dashboard template with user data
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	s.render(w, "dashboard.html", data)
}

func (s *Server) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			s.Logger.Println("No auth token found in cookies")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		tokenStr := cookie.Value
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			// remove the invalid token cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "auth_token",
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				Secure:   false, // ⚠️ set to true in production (HTTPS)
				SameSite: http.SameSiteLaxMode,
				Expires:  time.Now().Add(time.Hour),
			})
			s.Logger.Println("Invalid auth token:", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		user, ok := claims["user"].(string)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), userCtxKey, user)
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) LoginPage(w http.ResponseWriter, r *http.Request) {
	// check cookie for auth_token
	cookie, err := r.Cookie("auth_token")
	if err == nil && cookie != nil {
		// If token exists, redirect to dashboard
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	s.render(w, "login.html", map[string]any{
		"Title": "Login",
	})
}

func (s *Server) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Dummy check — add DB validation later
	if creds.Email != "marcus@example.com" || creds.Password != "secret" {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": creds.Email,
		"exp":  time.Now().Add(1 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		http.Error(w, "Could not create token", http.StatusInternalServerError)
		return
	}

	// ✅ Set token as secure, HttpOnly cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // ⚠️ set to true in production (HTTPS)
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(time.Hour),
	})
	// set cookie for CORS
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
	w.Header().Set("Content-Type", "text/plain")
	// Return the token as plain text
	w.WriteHeader(http.StatusOK)
	// Return the token as plain text	

	w.Write([]byte(tokenString))
}
// ListingPage handles displaying a list of listing or a specific listing by ID
// If no ID is provided, it lists all listings
func (s *Server) ListingPage(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/listings/")
	if path == "" || strings.Contains(path, "/") {
		ctx := r.Context()
		user, _ := ctx.Value(userCtxKey).(string)

		s.Logger.Println("Listing accessed by:", user)
		// fetch listings from the repository
		listings, err := s.repo.ListListings(ctx, repo.ListingFilter{
			// UserID: user, // Assuming UserID is stored in the context
			Limit: 10,
			Offset: 0,
			SortBy: "created_at",
		})
		if err != nil {
			s.Logger.Println("Error fetching listings:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		s.Logger.Println("No of listing:", len(listings))

		data := map[string]any{
			"Title":    "Listings",
			"Username": user,
			"Data": map[string]any{
				"Message": "Here are your listings!",
				"Listings": listings,
			},
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		s.render(w, "listing.html", data)
		return
	}

	idStr := path
	if idStr == "" {
	}

	ctx := r.Context()
	user, _ := ctx.Value(userCtxKey).(string)

	s.Logger.Println("GetListingPage accessed by:", user)

	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		http.Error(w, "Invalid listing ID", http.StatusBadRequest)
		return
	}

	listing, err := s.repo.GetListing(ctx, id)
	if err != nil {
		s.Logger.Println("Error fetching listing:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":    "Listing Details",
		"Username": user,
		"Data": map[string]any{
			"Message":  "Here are the details for the listing!",
			"Listing":  listing,
			"UserID":   listing.UserID,
			"Featured": listing.Featured,
			"CreatedAt": listing.CreatedAt,
		},
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	s.render(w, "listing_details.html", data)
}

// Add ListingHandler handles adding a new listing
func (s *Server) AddListingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the user from context
	ctx := r.Context()
	user, ok := ctx.Value(userCtxKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	s.Logger.Println("Adding listing for user:", user)

	var listing repo.Listing

	if err := json.NewDecoder(r.Body).Decode(&listing); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := s.repo.AddListing(ctx, listing); err != nil {
		http.Error(w, "Failed to add listing: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Listing added successfully"))
}

// Add listing form
func (s *Server) AddListingForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.Logger.Println("Rendering add listing form")

	data := map[string]any{
		"Title": "Add Listing",
	}
	s.render(w, "add_listing.html", data)
}

// GetListing retrieves a specific listing by ID
func (s *Server) GetListingPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := ctx.Value(userCtxKey).(string)

	s.Logger.Println("GetListingPage accessed by:", user)

	// Extract listing ID from URL query parameters
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Listing ID is required", http.StatusBadRequest)
		return
	}

	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		http.Error(w, "Invalid listing ID", http.StatusBadRequest)
		return
	}

	listing, err := s.repo.GetListing(ctx, id)
	if err != nil {
		s.Logger.Println("Error fetching listing:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":    "Listing Details",
		"Username": user,
		"Data": map[string]any{
			"Message":  "Here are the details for the listing!",
			"Listing":  listing,
			"UserID":   listing.UserID,
			"Featured": listing.Featured,
			"CreatedAt": listing.CreatedAt,
		},
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	s.render(w, "listing_details.html", data)
}

// BuyItemHandler
func (s *Server) BuyItemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	var creds struct {
		FullName string `json:"full_name"` // Full name of the buyer
		Email    string `json:"email"`
		Phone    string `json:"phone"` // Phone number of the buyer
		Message  string `json:"message"` // Additional message from the buyer
		ItemID   int    `json:"item_id"` // ID of the item being purchased
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	purchase := repo.Purchase{
		ItemID:       creds.ItemID,
		BuyerName:    creds.FullName,
		BuyerEmail:   creds.Email,
		BuyerPhone:   creds.Phone,
		BuyerAddress: "", // Optional, can be added later
		Metadata:     map[string]string{"purchase_message": creds.Message},
	}

	if err := s.repo.PurchaseItem(ctx, purchase); err != nil {
		s.Logger.Println("Error purchasing item:", err)
		http.Error(w, "Failed to purchase item: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Purchase started successfully!"))
	s.Logger.Printf("Purchase started for item ID %d by %s (%s)", creds.ItemID, creds.FullName, creds.Email)
}

// BuyItemPage
func (s *Server) BuyItemPage(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/buy/")
	if path == "" || strings.Contains(path, "/") {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	idStr := path
	if idStr == "" {
	}

	ctx := r.Context()
	user, _ := ctx.Value(userCtxKey).(string)

	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		http.Error(w, "Invalid listing ID", http.StatusBadRequest)
		return
	}

	listing, err := s.repo.GetListing(ctx, id)
	if err != nil {
		s.Logger.Println("Error fetching listing:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":    "Listing Details",
		"Username": user,
		"Data": map[string]any{
			"Message":  "Here are the details for the listing!",
			"Listing":  listing,
			"UserID":   listing.UserID,
			"Featured": listing.Featured,
			"CreatedAt": listing.CreatedAt,
		},
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	s.render(w, "buy_item.html", data)
}

// toJson helper
func toJson(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		log.Println("toJson error:", err)
		return "null"
	}
	return string(b)
}
