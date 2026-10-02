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
	"erp-azhan/api/internal/rbac"
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
	rbacRepo := rbac.NewRepository(db)
	rbacHandler := rbac.NewHandler(rbacRepo, identityRepo)
	// Loader RBAC untuk klaim JWT saat login/refresh (RBAC Fase 1).
	accessLoader := func(ctx context.Context, adminUserID int64) (*identity.UserAccess, error) {
		access, err := rbacRepo.GetUserAccess(ctx, adminUserID)
		if err != nil {
			return nil, err
		}
		return &identity.UserAccess{
			Roles:       access.Roles,
			Permissions: access.Permissions,
			PermVersion: access.PermVersion,
		}, nil
	}
	identityHandler := identity.NewHandler(identityRepo, accessLoader)

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
	crmUserHandler := crmuser.NewHandler(crmUserRepo, identityRepo, rbacRepo)

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
			r.Use(identity.RequirePermission("crmuser.admin"))
			r.Get("/", crmUserHandler.List)
			r.Post("/", crmUserHandler.Create)
			r.Put("/{id}", crmUserHandler.Update)
			r.Put("/{id}/password", crmUserHandler.ResetPassword)
		})

		// RBAC Fase 1: manajemen role & permission (SPEK-RBAC-2026-10-02.md).
		r.Route("/roles", func(r chi.Router) {
			r.Use(identity.RequirePermission("user.admin"))
			r.Get("/", rbacHandler.ListRoles)
		})
		r.With(identity.RequirePermission("user.admin")).Get("/permissions", rbacHandler.ListPermissions)
		r.Route("/users/{id}/roles", func(r chi.Router) {
			r.Use(identity.RequirePermission("user.admin"))
			r.Get("/", rbacHandler.GetUserRoles)
			r.Put("/", rbacHandler.SetUserRoles)
		})

		// RBAC Fase 2: guard route per permission (SPEK-RBAC-2026-10-02.md §8).
		// Pemetaan meniru akses yang berlaku sebelumnya; pengetatan kebijakan
		// adalah keputusan terpisah. Alias guard yang sering dipakai:
		masterView := r.With(identity.RequirePermission("masterdata.view"))
		masterEdit := r.With(identity.RequirePermission("masterdata.edit"))

		// Hotels
		masterView.Get("/hotels", hotelHandler.ListHotels)
		masterView.Get("/hotels/cities", hotelHandler.ListCities)
		masterEdit.Post("/hotels", hotelHandler.CreateHotel)
		masterEdit.Put("/hotels/{id}", hotelHandler.UpdateHotel)
		masterEdit.Delete("/hotels/{id}", hotelHandler.DeleteHotel)

		// Airlines
		masterView.Get("/airlines", airlineHandler.ListAirlines)
		masterEdit.Post("/airlines", airlineHandler.CreateAirline)
		masterEdit.Put("/airlines/{id}", airlineHandler.UpdateAirline)
		masterEdit.Delete("/airlines/{id}", airlineHandler.DeleteAirline)

		// Airports
		masterView.Get("/airports", airportHandler.ListAirports)
		masterEdit.Post("/airports", airportHandler.CreateAirport)
		masterEdit.Put("/airports/{id}", airportHandler.UpdateAirport)
		masterEdit.Delete("/airports/{id}", airportHandler.DeleteAirport)

		// Categories
		masterView.Get("/categories", categoryHandler.ListCategories)
		masterView.Get("/categories/{id}", categoryHandler.GetCategory)
		masterEdit.Post("/categories", categoryHandler.CreateCategory)
		masterEdit.Put("/categories/{id}", categoryHandler.UpdateCategory)
		masterEdit.Delete("/categories/{id}", categoryHandler.DeleteCategory)

		// Add-Ons
		masterView.Get("/addons", addonHandler.ListAddOns)
		masterEdit.Post("/addons", addonHandler.CreateAddOn)
		masterEdit.Put("/addons/{id}", addonHandler.UpdateAddOn)
		masterEdit.Delete("/addons/{id}", addonHandler.DeleteAddOn)

		r.Route("/brands", func(r chi.Router) {
			r.With(identity.RequirePermission("brand.view")).Get("/", brandHandler.ListBrands)
			r.With(identity.RequirePermission("brand.view")).Get("/{id}", brandHandler.GetBrand)
			r.With(identity.RequirePermission("brand.edit")).Post("/", brandHandler.CreateBrand)
			r.With(identity.RequirePermission("brand.edit")).Put("/{id}", brandHandler.UpdateBrand)
			r.With(identity.RequirePermission("brand.edit")).Delete("/{id}", brandHandler.DeleteBrand)
		})

		r.Route("/bank-accounts", func(r chi.Router) {
			r.With(identity.RequirePermission("bankaccount.view")).Get("/", bankAccountHandler.List)
			r.With(identity.RequirePermission("bankaccount.edit")).Post("/", bankAccountHandler.Create)
			r.With(identity.RequirePermission("bankaccount.edit")).Put("/{id}", bankAccountHandler.Update)
			r.With(identity.RequirePermission("bankaccount.edit")).Delete("/{id}", bankAccountHandler.Delete)
		})

		// Users: satu permission dengan manajemen role (user.admin).
		r.Route("/users", func(r chi.Router) {
			r.Use(identity.RequirePermission("user.admin"))
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
		masterView.Get("/itineraries", itineraryHandler.ListItineraries)
		masterView.Get("/itineraries/{id}", itineraryHandler.GetItinerary)
		masterEdit.Post("/itineraries", itineraryHandler.CreateItinerary)
		masterEdit.Put("/itineraries/{id}", itineraryHandler.UpdateItinerary)
		masterEdit.Delete("/itineraries/{id}", itineraryHandler.DeleteItinerary)

		// Schedules
		scheduleView := r.With(identity.RequirePermission("schedule.view"))
		scheduleEdit := r.With(identity.RequirePermission("schedule.edit"))
		scheduleView.Get("/schedules", scheduleHandler.ListSchedulesAdmin)
		scheduleView.Get("/schedules/{id}", scheduleHandler.GetScheduleAdmin)
		scheduleEdit.Post("/schedules", scheduleHandler.CreateSchedule)
		scheduleEdit.Put("/schedules/{id}", scheduleHandler.UpdateSchedule)
		scheduleEdit.Put("/schedules/{id}/status", scheduleHandler.UpdateScheduleStatus)
		scheduleEdit.Put("/schedules/{id}/seat", scheduleHandler.UpdateScheduleSeat)
		scheduleEdit.Delete("/schedules/{id}", scheduleHandler.DeleteSchedule)

		// Jamaah
		jamaahView := r.With(identity.RequirePermission("jamaah.view"))
		jamaahEdit := r.With(identity.RequirePermission("jamaah.edit"))
		jamaahView.Get("/jamaah", jamaahHandler.ListJamaah)
		jamaahView.Get("/jamaah/{id}", jamaahHandler.GetJamaah)
		jamaahEdit.Post("/jamaah", jamaahHandler.CreateJamaah)
		jamaahEdit.Put("/jamaah/{id}", jamaahHandler.UpdateJamaah)
		jamaahEdit.Put("/jamaah/{id}/catatan", jamaahHandler.UpdateCatatan)
		jamaahEdit.Delete("/jamaah/{id}", jamaahHandler.DeleteJamaah)
		jamaahEdit.Post("/jamaah/{id}/activation-link", portalHandler.GenerateActivationLink)

		// Relasi Kekerabatan Jamaah
		jamaahView.Get("/jamaah/{id}/relasi", jamaahHandler.ListRelasi)
		jamaahEdit.Post("/jamaah/{id}/relasi", jamaahHandler.CreateRelasi)
		jamaahEdit.Put("/jamaah/{id}/relasi/{relasi_id}", jamaahHandler.UpdateRelasi)
		jamaahEdit.Delete("/jamaah/{id}/relasi/{relasi_id}", jamaahHandler.DeleteRelasi)

		// Dokumen Jamaah
		r.With(identity.RequirePermission("dokumen.view")).Get("/jamaah/{jamaah_id}/dokumen", dokumenHandler.ListDokumen)
		r.With(identity.RequirePermission("dokumen.edit")).Post("/jamaah/{jamaah_id}/dokumen", dokumenHandler.UpsertDokumen)
		r.With(identity.RequirePermission("dokumen.edit")).Put("/dokumen/{id}/status", dokumenHandler.UpdateDokumenStatus)

		// Bookings
		bookingView := r.With(identity.RequirePermission("booking.view"))
		bookingEdit := r.With(identity.RequirePermission("booking.edit"))
		bookingView.Get("/bookings", bookingHandler.ListBookings)
		bookingView.Get("/bookings/{id}", bookingHandler.GetBooking)
		bookingEdit.Post("/bookings", bookingHandler.CreateBooking)
		bookingEdit.Delete("/bookings/{id}", bookingHandler.DeleteDraftBooking)
		bookingEdit.Post("/bookings/draft", bookingHandler.CreateDraftBooking)
		bookingEdit.Put("/bookings/{id}/draft", bookingHandler.UpdateDraftBooking)
		bookingEdit.Post("/bookings/{id}/finalize", bookingHandler.FinalizeBooking)
		bookingEdit.Put("/bookings/{id}/status", bookingHandler.UpdateBookingStatus)
		bookingEdit.Put("/bookings/{id}/seat-block", bookingHandler.BlockSeat)
		bookingEdit.Delete("/bookings/{id}/seat-block", bookingHandler.CancelSeatBlock)
		bookingEdit.Post("/bookings/{id}/addons", bookingHandler.AddBookingAddon)
		bookingEdit.Delete("/bookings/{id}/addons/{addon_id}", bookingHandler.DeleteBookingAddon)
		bookingEdit.Post("/bookings/{id}/discounts", bookingHandler.AddBookingDiscount)
		bookingEdit.Delete("/bookings/{id}/discounts/{discountID}", bookingHandler.RemoveBookingDiscount)
		bookingView.Get("/bookings/{id}/cashback", bookingHandler.GetKreditCashback)
		bookingEdit.Post("/bookings/{id}/cashback", bookingHandler.PakaiKreditCashback)
		r.With(identity.RequirePermission("refund.view")).Get("/bookings/{id}/refunds", bookingHandler.ListRefunds)
		r.With(identity.RequirePermission("refund.edit")).Post("/bookings/{id}/refunds", bookingHandler.CreateRefund)
		bookingEdit.Put("/bookings/{id}/progress", bookingHandler.UpdateBookingProgress)
		bookingEdit.Put("/bookings/{id}/pax/{pax_id}/progress", bookingHandler.UpdatePaxProgress)
		bookingEdit.Put("/bookings/{id}/pax/{pax_id}/cancel", bookingHandler.CancelPax)
		bookingEdit.Put("/bookings/{id}/pax/{pax_id}/room-type", bookingHandler.UpdatePaxRoomType)
		bookingEdit.Put("/bookings/{id}/pax/{pax_id}/perlengkapan/distribusi", bookingHandler.DistribusiPerlengkapan)
		bookingEdit.Delete("/bookings/{id}/pax/{pax_id}/perlengkapan/distribusi", bookingHandler.BatalkanPerlengkapan)

		// Perlengkapan Items (Global) — master data holding.
		r.With(identity.RequirePermission("perlengkapan.view")).Get("/perlengkapan-items", perlengkapanHandler.ListItems)
		masterEdit.Post("/perlengkapan-items", perlengkapanHandler.CreateItem)
		masterEdit.Put("/perlengkapan-items/{id}", perlengkapanHandler.UpdateItem)
		masterEdit.Delete("/perlengkapan-items/{id}", perlengkapanHandler.DeleteItem)

		// Perlengkapan Stok (Per Brand)
		r.With(identity.RequirePermission("perlengkapan.view")).Get("/perlengkapan-stok", perlengkapanHandler.ListStok)
		r.With(identity.RequirePermission("perlengkapan.edit")).Put("/perlengkapan-stok/{item_id}", perlengkapanHandler.UpdateStok)

		// Perlengkapan Set Template (Global) — master data holding.
		r.With(identity.RequirePermission("perlengkapan.view")).Get("/perlengkapan-set-template", perlengkapanHandler.GetSetTemplate)
		masterEdit.Put("/perlengkapan-set-template", perlengkapanHandler.UpdateSetTemplate)

		// Payments
		r.With(identity.RequirePermission("payment.view")).Get("/payments", paymentHandler.ListAllPayments)
		r.With(identity.RequirePermission("payment.view")).Get("/bookings/{booking_id}/payments", paymentHandler.ListPayments)
		r.With(identity.RequirePermission("payment.edit")).Post("/bookings/{booking_id}/payments", paymentHandler.CreatePayment)
		r.With(identity.RequirePermission("payment.verify")).Put("/payments/{id}/status", paymentHandler.UpdatePaymentStatus)
		r.With(identity.RequirePermission("payment.verify")).Delete("/payments/{id}", paymentHandler.DeletePayment)

		// CRM: konversi lead menjadi jamaah, booking, seat hold/payment dalam satu transaksi.
		r.With(identity.RequirePermission("crmdeal.process")).Post("/crm/deals", crmDealHandler.ProcessDeal)

		// Agen Syiar — Persetujuan Agen & status (screen B1/B2a). Akun CS sudah
		// diblokir oleh RequireAdminOrCRMAccess; brand scope dicek di repository.
		r.Route("/agen", func(r chi.Router) {
			r.Use(identity.RequirePermission("agen.view"))
			r.Get("/pengajuan", agenHandler.ListPengajuan)
			r.Get("/aktif", agenHandler.ListAgenAktif)
			r.Get("/", agenHandler.ListAgen)
			r.Get("/komisi", agenHandler.RiwayatKomisiAdmin)
			r.Get("/peringkat", agenHandler.PeringkatAgen)
			r.Get("/{jamaahID}", agenHandler.DetailAgen)
			r.With(identity.RequirePermission("agen.approve")).Post("/{jamaahID}/setujui", agenHandler.Setujui)
			r.With(identity.RequirePermission("agen.approve")).Post("/{jamaahID}/tolak", agenHandler.Tolak)
			r.With(identity.RequirePermission("agen.edit")).Put("/{jamaahID}/status", agenHandler.UbahStatus)
			r.With(identity.RequirePermission("agen.approve")).Post("/pembayaran/{id}/verifikasi", agenHandler.VerifikasiPembayaran)
			r.With(identity.RequirePermission("agen.approve")).Post("/pembayaran/{id}/tolak", agenHandler.TolakPembayaran)
			r.With(identity.RequirePermission("agen.approve")).Post("/pembayaran/{id}/bukti", agenHandler.UploadBuktiAdmin)
		})

		// Agen Syiar — Admin Master lintas brand: Persetujuan Pencairan (C1) dan
		// Ganti Kaitan Agen (C4). Permission khusus; Admin Travel tidak memilikinya.
		r.Group(func(r chi.Router) {
			r.Use(identity.RequirePermission("agen.payout.approve"))
			r.Get("/pencairan", agenHandler.ListPencairanAdmin)
			r.Post("/pencairan/{id}/setujui", agenHandler.SetujuiPencairan)
			r.Post("/pencairan/{id}/tolak", agenHandler.TolakPencairan)
		})
		r.Group(func(r chi.Router) {
			r.Use(identity.RequirePermission("agen.kaitan.edit"))
			r.Get("/kaitan-agen", agenHandler.CariJamaahKaitan)
			r.Get("/kaitan-agen/{jamaahID}/log", agenHandler.LogKaitan)
			r.Put("/kaitan-agen/{jamaahID}", agenHandler.GantiKaitan)
		})

		// Analytics lintas brand
		r.With(identity.RequirePermission("report.analytics.view")).Get("/analytics/transactions-30-days", paymentHandler.ListDailyBrandTransactions)
		r.With(identity.RequirePermission("report.analytics.view")).Get("/analytics/pax-30-days", bookingHandler.ListDailyBrandPax)
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
