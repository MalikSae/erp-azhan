package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	"erp-azhan/api/internal/addon"
	"erp-azhan/api/internal/adminuser"
	"erp-azhan/api/internal/airline"
	"erp-azhan/api/internal/bankaccount"
	"erp-azhan/api/internal/booking"
	"erp-azhan/api/internal/brand"
	"erp-azhan/api/internal/category"
	"erp-azhan/api/internal/dokumen"
	"erp-azhan/api/internal/hotel"
	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/itinerary"
	"erp-azhan/api/internal/jamaah"
	"erp-azhan/api/internal/media"
	"erp-azhan/api/internal/payment"
	"erp-azhan/api/internal/perlengkapan"
	"erp-azhan/api/internal/portal"
	"erp-azhan/api/internal/schedule"
	"erp-azhan/api/internal/shared"
)

func main() {
	// ─── Load .env ────────────────────────────────────────────────────────────
	if err := godotenv.Load(); err != nil {
		log.Println("[WARN] File .env tidak ditemukan, menggunakan environment system")
	}

	// ─── Config ───────────────────────────────────────────────────────────────
	cfg := shared.LoadConfig()

	// ─── Database ─────────────────────────────────────────────────────────────
	var db *sql.DB
	var dbErr error

	db, dbErr = shared.NewDB(cfg)
	if dbErr != nil {
		log.Printf("[WARN] Koneksi database gagal saat startup: %v. Server tetap berjalan (degraded mode)", dbErr)
		db = nil
	} else {
		defer db.Close()
		log.Printf("[INFO] Database terhubung: %s@%s:%s/%s", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)
	}

	// ─── Router & Middleware ──────────────────────────────────────────────────
	r := chi.NewRouter()

	allowedOrigins := []string{
		"http://localhost:5173",
		"http://localhost:5174",
		"http://localhost:3000",
		"https://localhost:3000",
		"http://*.azhan.test",
		"http://*.azhan.test:3000",
		"https://*.azhan.test",
		"https://*.azhan.test:3000",
		"http://*.test",
		"https://*.test",
	}
	if configured := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")); configured != "" {
		for _, origin := range strings.Split(configured, ",") {
			if origin = strings.TrimSpace(origin); origin != "" {
				allowedOrigins = append(allowedOrigins, origin)
			}
		}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// ─── Handlers ─────────────────────────────────────────────────────────────
	hotelRepo := hotel.NewRepository(db)
	hotelHandler := hotel.NewHandler(hotelRepo)

	airlineRepo := airline.NewRepository(db)
	airlineHandler := airline.NewHandler(airlineRepo)

	categoryRepo := category.NewRepository(db)
	categoryHandler := category.NewHandler(categoryRepo)

	addonRepo := addon.NewRepository(db)
	addonHandler := addon.NewHandler(addonRepo)

	itineraryRepo := itinerary.NewRepository(db)
	itineraryHandler := itinerary.NewHandler(itineraryRepo)

	scheduleRepo := schedule.NewRepository(db)
	scheduleHandler := schedule.NewHandler(scheduleRepo)

	identityRepo := identity.NewRepository(db)
	identityHandler := identity.NewHandler(identityRepo)

	brandRepo := brand.NewRepository(db)
	brandHandler := brand.NewHandler(brandRepo)
	bankAccountRepo := bankaccount.NewRepository(db)
	bankAccountHandler := bankaccount.NewHandler(bankAccountRepo)

	mediaHandler := media.NewHandler()

	jamaahRepo := jamaah.NewRepository(db)
	jamaahHandler := jamaah.NewHandler(jamaahRepo)

	bookingRepo := booking.NewRepository(db)
	bookingHandler := booking.NewHandler(bookingRepo)

	paymentRepo := payment.NewRepository(db)
	paymentHandler := payment.NewHandler(paymentRepo)

	dokumenRepo := dokumen.NewRepository(db)
	dokumenHandler := dokumen.NewHandler(dokumenRepo)

	perlengkapanRepo := perlengkapan.NewRepository(db)
	perlengkapanHandler := perlengkapan.NewHandler(perlengkapanRepo)

	adminuserRepo := adminuser.NewRepository(db)
	adminuserHandler := adminuser.NewHandler(adminuserRepo)

	portalHandler := portal.NewHandler(db, jamaahRepo, bookingRepo, paymentRepo, dokumenRepo)

	// ─── Routes ───────────────────────────────────────────────────────────────
	r.Get("/api/health", healthHandler(db))

	// Static files (public)
	fileServer := http.FileServer(http.Dir("./uploads"))
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", fileServer))

	// Public: jadwal yang sudah published (tanpa prefix /admin)
	r.With(requireDB(db)).Get("/api/schedules", scheduleHandler.ListSchedulesPublic)
	r.With(requireDB(db)).Get("/api/itineraries/{id}", itineraryHandler.GetPublicItinerary)

	// Public: categories & brand
	r.With(requireDB(db)).Get("/api/public/categories", categoryHandler.ListPublicCategories)
	r.With(requireDB(db)).Get("/api/public/brand", brandHandler.ResolveDomain)

	// Auth (public)
	r.Route("/api/auth", func(r chi.Router) {
		r.Use(requireDB(db)) // Cek DB sebelum proses auth
		r.Post("/login", identityHandler.Login)
		r.Post("/refresh", identityHandler.Refresh)
		r.Post("/logout", identityHandler.Logout)
	})

	// Portal Jamaah (login public + scoped portal endpoints)
	r.Route("/api/portal", func(r chi.Router) {
		r.Use(requireDB(db))
		r.Post("/login", portalHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(identity.RequirePortalAuth)
			r.Get("/me", portalHandler.GetMe)
			r.Get("/bookings", portalHandler.ListBookings)
			r.Get("/bookings/{id}", portalHandler.GetBookingByID)
			r.Get("/bookings/{id}/payments", portalHandler.ListPayments)
			r.Get("/bank-accounts", portalHandler.ListBankAccounts)
			r.Post("/bookings/{id}/payments", portalHandler.CreatePayment)
			r.Get("/dokumen", portalHandler.ListDokumen)
			r.Post("/dokumen", portalHandler.UploadDokumen)
			r.Post("/media/upload", mediaHandler.UploadPortalMedia)
		})
	})

	r.Route("/api/admin", func(r chi.Router) {
		// Guard: tolak semua admin request jika DB tidak tersedia, LALU validasi auth token JWT
		r.Use(requireDB(db))
		r.Use(identity.RequireAuth)

		r.Get("/my-brand", brandHandler.GetMyBrand)

		// Hotels
		r.Get("/hotels", hotelHandler.ListHotels)
		r.Get("/hotels/cities", hotelHandler.ListCities)
		r.Post("/hotels", hotelHandler.CreateHotel)
		r.Put("/hotels/{id}", hotelHandler.UpdateHotel)
		r.Delete("/hotels/{id}", hotelHandler.DeleteHotel)

		// Airlines
		r.Get("/airlines", airlineHandler.ListAirlines)
		r.Post("/airlines", airlineHandler.CreateAirline)
		r.Put("/airlines/{id}", airlineHandler.UpdateAirline)
		r.Delete("/airlines/{id}", airlineHandler.DeleteAirline)

		// Categories
		r.Get("/categories", categoryHandler.ListCategories)
		r.Get("/categories/{id}", categoryHandler.GetCategory)
		r.Post("/categories", categoryHandler.CreateCategory)
		r.Put("/categories/{id}", categoryHandler.UpdateCategory)
		r.Delete("/categories/{id}", categoryHandler.DeleteCategory)

		// Add-Ons
		r.Get("/addons", addonHandler.ListAddOns)
		r.Post("/addons", addonHandler.CreateAddOn)
		r.Put("/addons/{id}", addonHandler.UpdateAddOn)
		r.Delete("/addons/{id}", addonHandler.DeleteAddOn)

		r.Route("/brands", func(r chi.Router) {
			r.Use(brand.RequireSuperAdmin)
			r.Get("/", brandHandler.ListBrands)
			r.Get("/{id}", brandHandler.GetBrand)
			r.Post("/", brandHandler.CreateBrand)
			r.Put("/{id}", brandHandler.UpdateBrand)
			r.Delete("/{id}", brandHandler.DeleteBrand)
		})

		r.Route("/bank-accounts", func(r chi.Router) {
			r.Use(brand.RequireSuperAdmin)
			r.Get("/", bankAccountHandler.List)
			r.Post("/", bankAccountHandler.Create)
			r.Put("/{id}", bankAccountHandler.Update)
			r.Delete("/{id}", bankAccountHandler.Delete)
		})

		// Users (Super Admin Only)
		r.Route("/users", func(r chi.Router) {
			r.Use(brand.RequireSuperAdmin)
			r.Get("/", adminuserHandler.ListUsers)
			r.Post("/", adminuserHandler.CreateUser)
			r.Put("/{id}", adminuserHandler.UpdateUser)
			r.Put("/{id}/password", adminuserHandler.ResetPassword)
			r.Delete("/{id}", adminuserHandler.DeleteUser)
		})

		// Media
		r.Post("/media/upload", mediaHandler.UploadMedia)

		// Itineraries
		r.Get("/itineraries", itineraryHandler.ListItineraries)
		r.Get("/itineraries/{id}", itineraryHandler.GetItinerary)
		r.Post("/itineraries", itineraryHandler.CreateItinerary)
		r.Put("/itineraries/{id}", itineraryHandler.UpdateItinerary)
		r.Delete("/itineraries/{id}", itineraryHandler.DeleteItinerary)

		// Schedules
		r.Get("/schedules", scheduleHandler.ListSchedulesAdmin)
		r.Get("/schedules/{id}", scheduleHandler.GetScheduleAdmin)
		r.Post("/schedules", scheduleHandler.CreateSchedule)
		r.Put("/schedules/{id}", scheduleHandler.UpdateSchedule)
		r.Put("/schedules/{id}/status", scheduleHandler.UpdateScheduleStatus)
		r.Put("/schedules/{id}/seat", scheduleHandler.UpdateScheduleSeat)
		r.Delete("/schedules/{id}", scheduleHandler.DeleteSchedule)

		// Jamaah
		r.Get("/jamaah", jamaahHandler.ListJamaah)
		r.Get("/jamaah/{id}", jamaahHandler.GetJamaah)
		r.Post("/jamaah", jamaahHandler.CreateJamaah)
		r.Put("/jamaah/{id}", jamaahHandler.UpdateJamaah)
		r.Delete("/jamaah/{id}", jamaahHandler.DeleteJamaah)

		// Dokumen Jamaah
		r.Get("/jamaah/{jamaah_id}/dokumen", dokumenHandler.ListDokumen)
		r.Post("/jamaah/{jamaah_id}/dokumen", dokumenHandler.UpsertDokumen)
		r.Put("/dokumen/{id}/status", dokumenHandler.UpdateDokumenStatus)

		// Bookings
		r.Get("/bookings", bookingHandler.ListBookings)
		r.Get("/bookings/{id}", bookingHandler.GetBooking)
		r.Post("/bookings", bookingHandler.CreateBooking)
		r.Put("/bookings/{id}/status", bookingHandler.UpdateBookingStatus)
		r.Delete("/bookings/{id}/seat-block", bookingHandler.CancelSeatBlock)
		r.Post("/bookings/{id}/addons", bookingHandler.AddBookingAddon)
		r.Delete("/bookings/{id}/addons/{addon_id}", bookingHandler.DeleteBookingAddon)
		r.Put("/bookings/{id}/diskon", bookingHandler.UpdateBookingDiskon)
		r.Put("/bookings/{id}/progress", bookingHandler.UpdateBookingProgress)
		r.Put("/bookings/{id}/perlengkapan/distribusi", bookingHandler.DistribusiPerlengkapan)
		r.Delete("/bookings/{id}/perlengkapan/distribusi", bookingHandler.BatalkanPerlengkapan)

		// Perlengkapan Items (Global)
		r.Get("/perlengkapan-items", perlengkapanHandler.ListItems)
		r.Post("/perlengkapan-items", perlengkapanHandler.CreateItem)
		r.Put("/perlengkapan-items/{id}", perlengkapanHandler.UpdateItem)
		r.Delete("/perlengkapan-items/{id}", perlengkapanHandler.DeleteItem)

		// Perlengkapan Stok (Per Brand)
		r.Get("/perlengkapan-stok", perlengkapanHandler.ListStok)
		r.Put("/perlengkapan-stok/{item_id}", perlengkapanHandler.UpdateStok)

		// Perlengkapan Set Template (Global)
		r.Get("/perlengkapan-set-template", perlengkapanHandler.GetSetTemplate)
		r.Put("/perlengkapan-set-template", perlengkapanHandler.UpdateSetTemplate)

		// Payments
		r.Get("/payments", paymentHandler.ListAllPayments)
		r.Get("/bookings/{booking_id}/payments", paymentHandler.ListPayments)
		r.Post("/bookings/{booking_id}/payments", paymentHandler.CreatePayment)
		r.Put("/payments/{id}/status", paymentHandler.UpdatePaymentStatus)
		r.Delete("/payments/{id}", paymentHandler.DeletePayment)

		// Analytics lintas brand (Super Admin Only)
		r.With(brand.RequireSuperAdmin).Get("/analytics/transactions-30-days", paymentHandler.ListDailyBrandTransactions)
	})

	// ─── Start Server ─────────────────────────────────────────────────────────
	addr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Printf("[INFO] Server ERP Azhan API berjalan di http://localhost%s", addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("[ERROR] Server gagal berjalan: %v", err)
		os.Exit(1)
	}
}

// ─── Health Check Handler ─────────────────────────────────────────────────────

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

// healthHandler mengembalikan status koneksi database.
// Jika db nil (gagal koneksi saat startup) atau Ping gagal → 503.
// Jika Ping sukses → 200.
func healthHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		dbStatus := "connected"
		httpStatus := http.StatusOK

		if db == nil {
			dbStatus = "disconnected"
			httpStatus = http.StatusServiceUnavailable
		} else if err := db.Ping(); err != nil {
			log.Printf("[WARN] Health check: database ping gagal: %v", err)
			dbStatus = "disconnected"
			httpStatus = http.StatusServiceUnavailable
		}

		w.WriteHeader(httpStatus)
		json.NewEncoder(w).Encode(healthResponse{
			Status:   statusFromDB(dbStatus),
			Database: dbStatus,
		})
	}
}

// statusFromDB mengonversi status database ke status API.
func statusFromDB(dbStatus string) string {
	if dbStatus == "connected" {
		return "ok"
	}
	return "error"
}

// requireDB adalah middleware yang menolak request dengan 503
// jika koneksi database tidak tersedia (db nil atau Ping gagal).
func requireDB(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if db == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "database tidak tersedia",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
