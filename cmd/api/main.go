package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
	_ "time/tzdata" // zona waktu tersedia tanpa bergantung pada OS server

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	"erp-azhan/api/internal/addon"
	"erp-azhan/api/internal/adminuser"
	"erp-azhan/api/internal/agen"
	"erp-azhan/api/internal/airline"
	"erp-azhan/api/internal/airport"
	"erp-azhan/api/internal/bankaccount"
	"erp-azhan/api/internal/booking"
	"erp-azhan/api/internal/brand"
	"erp-azhan/api/internal/category"
	"erp-azhan/api/internal/crmdeal"
	"erp-azhan/api/internal/crmuser"
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
	"erp-azhan/api/internal/selfbooking"
	"erp-azhan/api/internal/shared"
)

func main() {
	// ─── Load .env ────────────────────────────────────────────────────────────
	if err := godotenv.Load(); err != nil {
		log.Println("[WARN] File .env tidak ditemukan, menggunakan environment system")
	}

	// ─── Config ───────────────────────────────────────────────────────────────
	cfg := shared.LoadConfig()
	// Zona waktu bisnis untuk time.Now(), JSON, dan koneksi DB (lihat shared.Config.DSN).
	time.Local = cfg.Location()
	log.Printf("[INFO] zona waktu aplikasi: %s", time.Local)
	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		if err := identity.ValidateJWTSecret(); err != nil {
			log.Fatalf("[ERROR] konfigurasi JWT tidak aman: %v", err)
		}
	}

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
	r.Use(middleware.RequestID)
	// middleware.RealIP sengaja tidak dipakai: ia menimpa RemoteAddr dari header
	// True-Client-IP/X-Forwarded-For tanpa cek proxy, sehingga rate limit bisa
	// diakali. IP client untuk rate limit diambil lewat shared.ClientIP.
	r.Use(securityHeaders)

	allowedOrigins := []string{}
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		allowedOrigins = append(allowedOrigins,
			"http://localhost:5173", "http://localhost:5174",
			"http://localhost:3000", "https://localhost:3000",
			"http://*.azhan.test", "http://*.azhan.test:3000",
			"https://*.azhan.test", "https://*.azhan.test:3000",
			"http://*.test", "https://*.test",
		)
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
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Brand-Id"},
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

	airportRepo := airport.NewRepository(db)
	airportHandler := airport.NewHandler(airportRepo)

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

	mediaHandler := media.NewHandler(db)

	jamaahRepo := jamaah.NewRepository(db)
	jamaahHandler := jamaah.NewHandler(jamaahRepo)

	bookingRepo := booking.NewRepository(db)
	bookingHandler := booking.NewHandler(bookingRepo)

	paymentRepo := payment.NewRepository(db)
	paymentHandler := payment.NewHandler(paymentRepo)
	crmDealRepo := crmdeal.NewRepository(db)
	crmDealHandler := crmdeal.NewHandler(crmDealRepo)
	crmUserRepo := crmuser.NewRepository(db)
	crmUserHandler := crmuser.NewHandler(crmUserRepo, identityRepo)

	selfBookingRepo := selfbooking.NewRepository(db)
	selfBookingHandler := selfbooking.NewHandler(selfBookingRepo)

	if db != nil {
		go runSeatHoldExpiry(crmDealRepo)
	}

	dokumenRepo := dokumen.NewRepository(db)
	dokumenHandler := dokumen.NewHandler(dokumenRepo)

	perlengkapanRepo := perlengkapan.NewRepository(db)
	perlengkapanHandler := perlengkapan.NewHandler(perlengkapanRepo)

	adminuserRepo := adminuser.NewRepository(db)
	adminuserHandler := adminuser.NewHandler(adminuserRepo, identityRepo)

	portalHandler := portal.NewHandler(db, jamaahRepo, bookingRepo, paymentRepo, dokumenRepo)

	agenRepo := agen.NewRepository(db)
	agenHandler := agen.NewHandler(agenRepo)

	// ─── Routes ───────────────────────────────────────────────────────────────
	r.Get("/api/health", healthHandler(db))

	// Branding media is public; sensitive documents are served only through the
	// authenticated /api/admin/media route below. Directory listing is never
	// enabled.
	r.Get("/uploads/{category}/{filename}", mediaHandler.ServePublic)

	// Public: jadwal yang sudah published (tanpa prefix /admin)
	r.With(requireDB(db)).Get("/api/schedules", scheduleHandler.ListSchedulesPublic)
	r.With(requireDB(db)).Get("/api/schedules/{id}", scheduleHandler.GetSchedulePublic)
	r.With(requireDB(db)).Get("/api/itineraries/{id}", itineraryHandler.GetPublicItinerary)

	// Public: categories & brand & bank accounts
	r.With(requireDB(db)).Get("/api/public/categories", categoryHandler.ListPublicCategories)
	r.With(requireDB(db)).Get("/api/public/brand", brandHandler.ResolveDomain)
	r.With(requireDB(db)).Get("/api/public/bank-accounts", bankAccountHandler.List)

	// Public: Booking (Self-Service) & Digital Invoice & Jamaah Check
	r.With(requireDB(db)).Post("/api/public/book", selfBookingHandler.CreateBooking)
	r.With(requireDB(db)).Post("/api/public/agen/daftar", selfBookingHandler.DaftarAgen)
	r.With(requireDB(db)).Get("/api/public/invoice/{code}", selfBookingHandler.GetPublicInvoice)
	r.With(requireDB(db)).Post("/api/public/jamaah/check", selfBookingHandler.CheckPhone)
	r.With(requireDB(db)).Post("/api/public/aktivasi/check", portalHandler.CheckActivationToken)
	r.With(requireDB(db)).Post("/api/public/aktivasi/verify-dob", portalHandler.VerifyActivationDob)
	r.With(requireDB(db)).Post("/api/public/aktivasi", portalHandler.ActivateAccount)

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
			r.Use(identity.RequirePortalSession(db))
			r.Post("/logout", identity.PortalLogout(db))
			r.Get("/me", portalHandler.GetMe)
			r.Get("/cashback", portalHandler.GetCashback)
			r.Get("/bookings", portalHandler.ListBookings)
			r.Get("/bookings/{id}", portalHandler.GetBookingByID)
			r.Get("/bookings/{id}/payments", portalHandler.ListPayments)
			r.Get("/bookings/{id}/invoice-link", selfBookingHandler.InvoiceLink)
			r.Get("/bank-accounts", portalHandler.ListBankAccounts)
			r.Post("/bookings/{id}/payments", portalHandler.CreatePayment)
			r.Get("/dokumen", portalHandler.ListDokumen)
			r.Get("/dokumen/{id}/file", mediaHandler.ServeDocument)
			r.Post("/dokumen", portalHandler.UploadDokumen)
			r.Post("/media/upload", mediaHandler.UploadPortalMedia)

			// Agen Syiar (screen A1–A4)
			r.Get("/agen", agenHandler.GetStatus)
			r.Post("/agen/pengajuan", agenHandler.Ajukan)
			r.Post("/agen/pembayaran/bukti", agenHandler.UploadBukti)

			// Agen Syiar aktif (screen A3, A5, A6)
			r.Get("/agen/dashboard", agenHandler.Dashboard)
			r.Get("/agen/komisi", agenHandler.RiwayatKomisiPortal)
			r.Get("/agen/jamaah", selfBookingHandler.ListJamaahSaya)
			r.Post("/agen/bookings", selfBookingHandler.CreateBookingAgen)

			// Agen Syiar: tarik saldo (screen A7)
			r.Get("/agen/pencairan", agenHandler.GetPencairan)
			r.Post("/agen/pencairan", agenHandler.AjukanPencairan)
			r.Get("/agen/pencairan/{id}/bukti", agenHandler.BuktiPencairanPortal)
		})
	})

	r.Route("/api/admin", func(r chi.Router) {
		// Guard: tolak semua admin request jika DB tidak tersedia, LALU validasi auth token JWT
		r.Use(requireDB(db))
		r.Use(identity.RequireAuth)
		r.Use(identity.RequireAdminOrCRMAccess)

		r.Get("/my-brand", brandHandler.GetMyBrand)
		r.With(identity.RequireAdminRole).Put("/account/password", adminuserHandler.ChangeOwnPassword)

		r.Route("/crm/users", func(r chi.Router) {
			r.Use(identity.RequireAdminRole)
			r.Get("/", crmUserHandler.List)
			r.Post("/", crmUserHandler.Create)
			r.Put("/{id}", crmUserHandler.Update)
			r.Put("/{id}/password", crmUserHandler.ResetPassword)
		})

		// Master data global milik holding: semua admin boleh baca,
		// hanya Super Admin Grup yang boleh tambah/ubah/hapus.
		superAdmin := r.With(brand.RequireSuperAdmin)

		// Hotels
		r.Get("/hotels", hotelHandler.ListHotels)
		r.Get("/hotels/cities", hotelHandler.ListCities)
		superAdmin.Post("/hotels", hotelHandler.CreateHotel)
		superAdmin.Put("/hotels/{id}", hotelHandler.UpdateHotel)
		superAdmin.Delete("/hotels/{id}", hotelHandler.DeleteHotel)

		// Airlines
		r.Get("/airlines", airlineHandler.ListAirlines)
		superAdmin.Post("/airlines", airlineHandler.CreateAirline)
		superAdmin.Put("/airlines/{id}", airlineHandler.UpdateAirline)
		superAdmin.Delete("/airlines/{id}", airlineHandler.DeleteAirline)

		// Airports
		r.Get("/airports", airportHandler.ListAirports)
		superAdmin.Post("/airports", airportHandler.CreateAirport)
		superAdmin.Put("/airports/{id}", airportHandler.UpdateAirport)
		superAdmin.Delete("/airports/{id}", airportHandler.DeleteAirport)

		// Categories
		r.Get("/categories", categoryHandler.ListCategories)
		r.Get("/categories/{id}", categoryHandler.GetCategory)
		r.Post("/categories", categoryHandler.CreateCategory)
		r.Put("/categories/{id}", categoryHandler.UpdateCategory)
		r.Delete("/categories/{id}", categoryHandler.DeleteCategory)

		// Add-Ons
		r.Get("/addons", addonHandler.ListAddOns)
		superAdmin.Post("/addons", addonHandler.CreateAddOn)
		superAdmin.Put("/addons/{id}", addonHandler.UpdateAddOn)
		superAdmin.Delete("/addons/{id}", addonHandler.DeleteAddOn)

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
		r.Get("/media/{category}/{filename}", mediaHandler.ServeProtected)
		r.Post("/media/upload", mediaHandler.UploadMedia)

		// Itineraries
		r.Get("/itineraries", itineraryHandler.ListItineraries)
		r.Get("/itineraries/{id}", itineraryHandler.GetItinerary)
		superAdmin.Post("/itineraries", itineraryHandler.CreateItinerary)
		superAdmin.Put("/itineraries/{id}", itineraryHandler.UpdateItinerary)
		superAdmin.Delete("/itineraries/{id}", itineraryHandler.DeleteItinerary)

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
		r.Put("/jamaah/{id}/catatan", jamaahHandler.UpdateCatatan)
		r.Delete("/jamaah/{id}", jamaahHandler.DeleteJamaah)
		r.Post("/jamaah/{id}/activation-link", portalHandler.GenerateActivationLink)

		// Relasi Kekerabatan Jamaah
		r.Get("/jamaah/{id}/relasi", jamaahHandler.ListRelasi)
		r.Post("/jamaah/{id}/relasi", jamaahHandler.CreateRelasi)
		r.Put("/jamaah/{id}/relasi/{relasi_id}", jamaahHandler.UpdateRelasi)
		r.Delete("/jamaah/{id}/relasi/{relasi_id}", jamaahHandler.DeleteRelasi)

		// Dokumen Jamaah
		r.Get("/jamaah/{jamaah_id}/dokumen", dokumenHandler.ListDokumen)
		r.Post("/jamaah/{jamaah_id}/dokumen", dokumenHandler.UpsertDokumen)
		r.Put("/dokumen/{id}/status", dokumenHandler.UpdateDokumenStatus)

		// Bookings
		r.Get("/bookings", bookingHandler.ListBookings)
		r.Get("/bookings/{id}", bookingHandler.GetBooking)
		r.Post("/bookings", bookingHandler.CreateBooking)
		r.Delete("/bookings/{id}", bookingHandler.DeleteDraftBooking)
		r.Post("/bookings/draft", bookingHandler.CreateDraftBooking)
		r.Put("/bookings/{id}/draft", bookingHandler.UpdateDraftBooking)
		r.Post("/bookings/{id}/finalize", bookingHandler.FinalizeBooking)
		r.Put("/bookings/{id}/status", bookingHandler.UpdateBookingStatus)
		r.Put("/bookings/{id}/seat-block", bookingHandler.BlockSeat)
		r.Delete("/bookings/{id}/seat-block", bookingHandler.CancelSeatBlock)
		r.Post("/bookings/{id}/addons", bookingHandler.AddBookingAddon)
		r.Delete("/bookings/{id}/addons/{addon_id}", bookingHandler.DeleteBookingAddon)
		r.Post("/bookings/{id}/discounts", bookingHandler.AddBookingDiscount)
		r.Delete("/bookings/{id}/discounts/{discountID}", bookingHandler.RemoveBookingDiscount)
		r.With(identity.RequireAdminRole).Get("/bookings/{id}/cashback", bookingHandler.GetKreditCashback)
		r.With(identity.RequireAdminRole).Post("/bookings/{id}/cashback", bookingHandler.PakaiKreditCashback)
		r.With(identity.RequireAdminRole).Get("/bookings/{id}/refunds", bookingHandler.ListRefunds)
		r.With(identity.RequireAdminRole).Post("/bookings/{id}/refunds", bookingHandler.CreateRefund)
		r.Put("/bookings/{id}/progress", bookingHandler.UpdateBookingProgress)
		r.Put("/bookings/{id}/pax/{pax_id}/progress", bookingHandler.UpdatePaxProgress)
		r.Put("/bookings/{id}/pax/{pax_id}/cancel", bookingHandler.CancelPax)
		r.Put("/bookings/{id}/pax/{pax_id}/room-type", bookingHandler.UpdatePaxRoomType)
		r.Put("/bookings/{id}/pax/{pax_id}/perlengkapan/distribusi", bookingHandler.DistribusiPerlengkapan)
		r.Delete("/bookings/{id}/pax/{pax_id}/perlengkapan/distribusi", bookingHandler.BatalkanPerlengkapan)

		// Perlengkapan Items (Global)
		r.Get("/perlengkapan-items", perlengkapanHandler.ListItems)
		superAdmin.Post("/perlengkapan-items", perlengkapanHandler.CreateItem)
		superAdmin.Put("/perlengkapan-items/{id}", perlengkapanHandler.UpdateItem)
		superAdmin.Delete("/perlengkapan-items/{id}", perlengkapanHandler.DeleteItem)

		// Perlengkapan Stok (Per Brand)
		r.Get("/perlengkapan-stok", perlengkapanHandler.ListStok)
		r.Put("/perlengkapan-stok/{item_id}", perlengkapanHandler.UpdateStok)

		// Perlengkapan Set Template (Global)
		r.Get("/perlengkapan-set-template", perlengkapanHandler.GetSetTemplate)
		superAdmin.Put("/perlengkapan-set-template", perlengkapanHandler.UpdateSetTemplate)

		// Payments
		r.Get("/payments", paymentHandler.ListAllPayments)
		r.Get("/bookings/{booking_id}/payments", paymentHandler.ListPayments)
		r.Post("/bookings/{booking_id}/payments", paymentHandler.CreatePayment)
		r.Put("/payments/{id}/status", paymentHandler.UpdatePaymentStatus)
		r.Delete("/payments/{id}", paymentHandler.DeletePayment)

		// CRM: konversi lead menjadi jamaah, booking, seat hold/payment dalam satu transaksi.
		r.Post("/crm/deals", crmDealHandler.ProcessDeal)

		// Agen Syiar — Persetujuan Agen & status (screen B1/B2a). Akun CS sudah
		// diblokir oleh RequireAdminOrCRMAccess; brand scope dicek di repository.
		r.Route("/agen", func(r chi.Router) {
			r.Use(identity.RequireAdminRole)
			r.Get("/pengajuan", agenHandler.ListPengajuan)
			r.Get("/aktif", agenHandler.ListAgenAktif)
			r.Get("/", agenHandler.ListAgen)
			r.Get("/komisi", agenHandler.RiwayatKomisiAdmin)
			r.Get("/{jamaahID}", agenHandler.DetailAgen)
			r.Post("/{jamaahID}/setujui", agenHandler.Setujui)
			r.Post("/{jamaahID}/tolak", agenHandler.Tolak)
			r.Put("/{jamaahID}/status", agenHandler.UbahStatus)
			r.Post("/pembayaran/{id}/verifikasi", agenHandler.VerifikasiPembayaran)
			r.Post("/pembayaran/{id}/tolak", agenHandler.TolakPembayaran)
			r.Post("/pembayaran/{id}/bukti", agenHandler.UploadBuktiAdmin)
		})

		// Agen Syiar — Admin Master lintas brand: Persetujuan Pencairan (C1) dan
		// Ganti Kaitan Agen (C4). Admin Travel tidak punya akses.
		r.Group(func(r chi.Router) {
			r.Use(identity.RequireAdminRole, brand.RequireSuperAdmin)
			r.Get("/pencairan", agenHandler.ListPencairanAdmin)
			r.Post("/pencairan/{id}/setujui", agenHandler.SetujuiPencairan)
			r.Post("/pencairan/{id}/tolak", agenHandler.TolakPencairan)
			r.Get("/kaitan-agen", agenHandler.CariJamaahKaitan)
			r.Get("/kaitan-agen/{jamaahID}/log", agenHandler.LogKaitan)
			r.Put("/kaitan-agen/{jamaahID}", agenHandler.GantiKaitan)
		})

		// Analytics lintas brand (Super Admin Only)
		r.With(brand.RequireSuperAdmin).Get("/analytics/transactions-30-days", paymentHandler.ListDailyBrandTransactions)
		r.With(brand.RequireSuperAdmin).Get("/analytics/pax-30-days", bookingHandler.ListDailyBrandPax)
	})

	// ─── Start Server ─────────────────────────────────────────────────────────
	bindHost := strings.TrimSpace(os.Getenv("APP_BIND_HOST"))
	if bindHost == "" {
		bindHost = "127.0.0.1"
	}
	addr := net.JoinHostPort(bindHost, cfg.AppPort)
	log.Printf("[INFO] Server ERP Azhan API berjalan di http://localhost%s", addr)

	server := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("[ERROR] Server gagal berjalan: %v", err)
		os.Exit(1)
	}
}

// securityHeaders applies conservative browser protections to every response.
// CSP is intentionally left to the reverse proxy because the API serves JSON
// and is consumed by multiple frontends.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func runSeatHoldExpiry(repo *crmdeal.Repository) {
	release := func() {
		count, err := repo.ReleaseExpiredSeatHolds(context.Background())
		if err != nil {
			log.Printf("[WARN] Gagal melepas seat hold kedaluwarsa: %v", err)
			return
		}
		if count > 0 {
			log.Printf("[INFO] Seat hold kedaluwarsa dilepas: %d booking", count)
		}
	}
	release()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		release()
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
