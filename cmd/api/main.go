package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // zona waktu tersedia tanpa bergantung pada OS server

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	"erp-azhan/api/internal/adminuser"
	"erp-azhan/api/internal/agen"
	"erp-azhan/api/internal/airline"
	"erp-azhan/api/internal/airport"
	"erp-azhan/api/internal/analitik"
	"erp-azhan/api/internal/bankaccount"
	"erp-azhan/api/internal/banner"
	"erp-azhan/api/internal/benefit"
	"erp-azhan/api/internal/booking"
	"erp-azhan/api/internal/brand"
	"erp-azhan/api/internal/category"
	"erp-azhan/api/internal/customer"
	"erp-azhan/api/internal/dokumen"
	"erp-azhan/api/internal/haramain"
	"erp-azhan/api/internal/hotel"
	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/itinerary"
	"erp-azhan/api/internal/jamaah"
	"erp-azhan/api/internal/mailer"
	"erp-azhan/api/internal/manasik"
	"erp-azhan/api/internal/media"
	"erp-azhan/api/internal/notifikasi"
	"erp-azhan/api/internal/tabungan"
	"erp-azhan/api/internal/payment"
	"erp-azhan/api/internal/pengumuman"
	"erp-azhan/api/internal/perlengkapan"
	"erp-azhan/api/internal/akses"
	"erp-azhan/api/internal/manifest"
	"erp-azhan/api/internal/schedule"
	"erp-azhan/api/internal/selfbooking"
	"erp-azhan/api/internal/shared"
	"erp-azhan/api/internal/surat"
	"erp-azhan/api/internal/topbar"
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
	// JWT_SECRET divalidasi SELALU (semua APP_ENV): kosong, < 32 karakter, atau placeholder = gagal start.
	if err := identity.ValidateJWTSecret(); err != nil {
		log.Fatalf("[ERROR] konfigurasi JWT tidak aman: %v", err)
	}
	if mins, _ := strconv.Atoi(strings.TrimSpace(os.Getenv("JWT_ACCESS_TTL_MINUTES"))); mins > 60 {
		log.Printf("[WARN] JWT_ACCESS_TTL_MINUTES=%d dibatasi menjadi 60 menit", mins)
	}
	// Captcha opsional: tanpa TURNSTILE_SECRET_KEY nonaktif (hanya dicatat).
	if err := shared.ValidateCaptchaConfig(); err != nil {
		log.Fatalf("[ERROR] konfigurasi captcha: %v", err)
	}
	// Aktivasi akun jamaah: bawaan mati (rute 404). Bila menyala, pepper minimal 32 byte (fail closed).
	aktivasiCfg, err := customer.KonfigAktivasiDariEnv()
	if err != nil {
		log.Fatalf("[ERROR] konfigurasi aktivasi akun jamaah: %v", err)
	}
	if !shared.IsProduction() {
		log.Printf("[WARN] APP_ENV=%q (bukan production): origin CORS pengembangan aktif. Server publik WAJIB APP_ENV=production.", shared.AppEnv())
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
	// Batas body 1 MB untuk semua request; unggahan berkas memasang batas 10 MB sendiri di paket media.
	r.Use(shared.LimitBody(shared.MaxJSONBodyBytes, "/api/admin/media/upload", "/api/account/media/upload"))

	allowedOrigins := corsOrigins(shared.IsProduction(), os.Getenv("CORS_ALLOWED_ORIGINS"))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Brand-Id"},
		ExposedHeaders:   []string{"Link", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Logger akses dengan token invoice/token query disamarkan (shared.RedactURI).
	r.Use(shared.RedactingLogger(&middleware.DefaultLogFormatter{Logger: log.New(os.Stdout, "", log.LstdFlags), NoColor: false}))
	r.Use(middleware.Recoverer)

	// ─── Handlers ─────────────────────────────────────────────────────────────
	hotelRepo := hotel.NewRepository(db)
	hotelHandler := hotel.NewHandler(hotelRepo)

	airlineRepo := airline.NewRepository(db)
	airlineHandler := airline.NewHandler(airlineRepo)
	benefitHandler := benefit.NewHandler(benefit.NewRepository(db))

	airportRepo := airport.NewRepository(db)
	airportHandler := airport.NewHandler(airportRepo)

	categoryRepo := category.NewRepository(db)
	categoryHandler := category.NewHandler(categoryRepo)

	itineraryRepo := itinerary.NewRepository(db)
	itineraryHandler := itinerary.NewHandler(itineraryRepo)

	scheduleRepo := schedule.NewRepository(db)
	scheduleHandler := schedule.NewHandler(scheduleRepo)
	manifestHandler := manifest.NewHandler(db)
	// Paket yang tanggal berangkatnya sudah lewat diarsipkan otomatis: sekali saat API menyala, lalu tiap 24 jam.
	go schedule.JalankanArsipOtomatis(context.Background(), scheduleRepo, 24*time.Hour)

	identityRepo := identity.NewRepository(db)
	aksesRepo := akses.NewRepository(db)
	// Loader akses untuk klaim JWT saat login/refresh (SPEK-AKSES-PENGGUNA-2026-11-25).
	accessLoader := func(ctx context.Context, adminUserID int64) (*identity.UserAccess, error) {
		access, err := aksesRepo.GetUserAccess(ctx, adminUserID)
		if err != nil {
			return nil, err
		}
		return &identity.UserAccess{
			Role:        access.Jenis,
			Roles:       []string{access.Jenis},
			Permissions: access.Permissions,
			PermVersion: access.PermVersion,
		}, nil
	}
	identityHandler := identity.NewHandler(identityRepo, accessLoader)
	if db != nil {
		// RequireAuth memeriksa is_active/perm_version (cache 30 detik) agar pencabutan akses cepat berlaku.
		identity.SetUserStatusLoader(identityRepo.UserStatus)
	}

	brandRepo := brand.NewRepository(db)
	brandHandler := brand.NewHandler(brandRepo)
	brandHandler.SetAktivasiJamaah(aktivasiCfg.Aktif)
	suratHandler := surat.NewHandler(surat.NewRepository(db))
	bankAccountRepo := bankaccount.NewRepository(db)
	bankAccountHandler := bankaccount.NewHandler(bankAccountRepo)

	mediaHandler := media.NewHandler(db)

	jamaahRepo := jamaah.NewRepository(db)
	jamaahHandler := jamaah.NewHandler(jamaahRepo)

	bookingRepo := booking.NewRepository(db)
	bookingHandler := booking.NewHandler(bookingRepo)

	paymentRepo := payment.NewRepository(db)
	paymentHandler := payment.NewHandler(paymentRepo)

	selfBookingRepo := selfbooking.NewRepository(db)
	selfBookingHandler := selfbooking.NewHandler(selfBookingRepo, mailer.NewFromEnv())

	if db != nil {
		go runSeatHoldExpiry(bookingRepo)
		go runAktivasiCleanup(db)
		go notifikasi.JalankanPengingat(context.Background(), db, time.Minute)
		if v := strings.ToLower(strings.TrimSpace(os.Getenv("PELUNASAN_AUTO_BATAL"))); v == "false" || v == "0" || v == "off" {
			log.Printf("[INFO] pembatalan otomatis lewat batas pelunasan DIMATIKAN (PELUNASAN_AUTO_BATAL=%s)", v)
		} else {
			go runPelunasanExpiry(bookingRepo, db)
		}
	}

	dokumenRepo := dokumen.NewRepository(db)
	dokumenHandler := dokumen.NewHandler(dokumenRepo)

	perlengkapanRepo := perlengkapan.NewRepository(db)
	perlengkapanHandler := perlengkapan.NewHandler(perlengkapanRepo)

	adminuserRepo := adminuser.NewRepository(db)
	adminuserHandler := adminuser.NewHandler(adminuserRepo, identityRepo)

	customerRepo := customer.NewRepository(db)
	customerHandler := customer.NewHandler(customerRepo, mailer.NewFromEnv())
	jamaahAkunHandler := customer.NewJamaahAkun(db, customerRepo, mailer.NewFromEnv())
	aktivasiHandler := customer.NewAktivasiHandler(customer.NewAktivasi(db, customerRepo, aktivasiCfg.Pepper), customerRepo, aktivasiCfg)
	pengumumanHandler := pengumuman.NewHandler(db)
	manasikHandler := manasik.NewHandler(db)
	bannerHandler := banner.NewHandler(db)
	notifikasiHandler := notifikasi.NewHandler(db)
	tabunganHandler := tabungan.NewHandler(db)
	haramainHandler := haramain.NewHandler(db)
	topbarHandler := topbar.NewHandler(db)
	customerOrders := customer.NewOrders(db, bookingRepo, paymentRepo)

	agenRepo := agen.NewRepository(db)
	agenHandler := agen.NewHandler(agenRepo)
	analitikHandler := analitik.NewHandler(analitik.NewRepository(db, agenRepo))
	adminAkunHandler := customer.NewAdminAkun(db, agenRepo)

	// ─── Routes ───────────────────────────────────────────────────────────────
	r.Get("/api/health", healthHandler(db))

	// Branding media is public; sensitive documents are served only through the
	// authenticated /api/admin/media route below. Directory listing is never
	// enabled.
	r.Get("/uploads/{category}/{filename}", mediaHandler.ServePublic)

	// Public: jadwal yang sudah published (tanpa prefix /admin)
	r.With(requireDB(db)).Get("/api/schedules", scheduleHandler.ListSchedulesPublic)
	r.With(requireDB(db)).Get("/api/schedules/{id}", scheduleHandler.GetSchedulePublic)
	// API publik v1 (read-only, stabil) untuk situs publik dan CMS: SPEK-WEB-PUBLIK-CMS-2026-10-20.md §4.3.
	// Tidak berisi data per pengguna: boleh di-cache publik 60 detik (hanya jawaban 200).
	publicV1 := r.With(requireDB(db), shared.PublicCache(60))
	publicV1.Get("/api/public/v1/packages", scheduleHandler.ListPackagesV1)
	publicV1.Get("/api/public/v1/packages/{id}", scheduleHandler.GetPackageV1)
	publicV1.Get("/api/public/v1/sitemap-feed", scheduleHandler.SitemapFeedV1)
	// Rentang komisi agen dari jadwal tayang untuk landing /jadi-agen (spek keagenan K23).
	publicV1.Get("/api/public/v1/agen/ujroh", agenHandler.UjrohPublik)
	// Banner Promosi Beranda app (maks 5 banner tayang).
	publicV1.Get("/api/public/v1/promosi", bannerHandler.Promosi)
	r.With(requireDB(db)).Post("/api/public/promosi/{id}/klik", bannerHandler.Klik)
	// Jelajah Haramain (web dan app). Detail tanpa cache publik supaya ulasan baru langsung terlihat.
	publicV1.Get("/api/public/v1/haramain", haramainHandler.DaftarPublik)
	r.With(requireDB(db)).Get("/api/public/v1/haramain/{id}", haramainHandler.DetailPublik)
	r.With(requireDB(db)).Get("/api/itineraries/{id}", itineraryHandler.GetPublicItinerary)

	// Public: categories & brand & bank accounts
	r.With(requireDB(db)).Get("/api/public/categories", categoryHandler.ListPublicCategories)
	r.With(requireDB(db)).Get("/api/public/brand", brandHandler.ResolveDomain)
	// Publik: wajib brand, hanya rekening aktif dan field untuk pelanggan (handler admin terpisah).
	r.With(requireDB(db)).Get("/api/public/bank-accounts", bankAccountHandler.ListPublic)
	// Klik link referral agen dari situs publik (satu per pengunjung per hari).
	r.With(requireDB(db)).Post("/api/public/referral/klik", agenHandler.KlikReferral)

	// Akun pelanggan (email + password): daftar, masuk, lupa password
	r.Route("/api/public/auth", func(r chi.Router) {
		r.Use(requireDB(db))
		r.Post("/register", customerHandler.Register)
		r.Post("/register-agen", customerHandler.RegisterAgen)
		r.Post("/register-app", customerHandler.RegisterApp)
		r.Post("/aktivasi", aktivasiHandler.Publik)
		r.Post("/login", customerHandler.Login)
		r.Post("/google", customerHandler.LoginGoogle)
		r.Post("/google/daftar", customerHandler.DaftarGoogle)
		r.Post("/forgot-password", customerHandler.ForgotPassword)
		r.Post("/reset-password", customerHandler.ResetPassword)
	})

	// Auth (public)
	r.Route("/api/auth", func(r chi.Router) {
		r.Use(requireDB(db)) // Cek DB sebelum proses auth
		r.Post("/login", identityHandler.Login)
		r.Post("/refresh", identityHandler.Refresh)
		r.Post("/logout", identityHandler.Logout)
	})

	// Area agen Syiar berbasis akun email (spek keagenan bagian 10 butir 10). Sesi sama dengan /api/account.
	// Status dan pengajuan terbuka untuk akun mana pun; selebihnya hanya agen aktif (agen nonaktif 403, layar A4).
	// Foto dan bukti bayar diunggah lewat POST /api/account/media/upload, lalu URL-nya dikirim ke sini.
	r.Route("/api/portal/agen", func(r chi.Router) {
		r.Use(requireDB(db))
		r.Use(customerHandler.RequireCustomer)
		r.Get("/", agenHandler.GetStatus)
		r.Post("/pengajuan", agenHandler.Ajukan)
		r.Post("/pembayaran/bukti", agenHandler.UploadBukti)
		r.Group(func(r chi.Router) {
			r.Use(agenHandler.RequireAgenAktif)
			r.Get("/dashboard", agenHandler.Dashboard)
			r.Get("/komisi", agenHandler.RiwayatKomisiPortal)
			r.Get("/mutasi", agenHandler.MutasiPortal)
			r.Get("/data-jamaah", agenHandler.DataJamaahPortal)
			r.Get("/data-jamaah/{bookingID}/{jamaahID}", agenHandler.DetailJamaahPortal)
			r.Get("/binaan", agenHandler.AgenBinaanPortal)
			r.Get("/binaan/{akunID}", agenHandler.DetailBinaanPortal)
			r.Get("/binaan/{akunID}/bonus", agenHandler.RincianBonusPortal)
			r.Get("/komisi-paket", agenHandler.KomisiPaketPortal)
			r.Get("/peringkat", agenHandler.PeringkatPortal)
			r.Get("/peringkat/{akunID}/foto", agenHandler.FotoPeringkatPortal)
			r.Get("/pencairan", agenHandler.GetPencairan)
			r.Post("/pencairan", agenHandler.AjukanPencairan)
			r.Get("/pencairan/{id}/bukti", agenHandler.BuktiPencairanPortal)
			r.Get("/jamaah", selfBookingHandler.ListJamaahSaya)
			r.Post("/bookings", selfBookingHandler.CreateBookingAgen)
		})
	})

	// Area akun pelanggan: butuh sesi (Authorization: Bearer <token sesi>).
	r.Route("/api/account", func(r chi.Router) {
		r.Use(requireDB(db))
		r.Use(customerHandler.RequireCustomer)
		r.Get("/me", customerHandler.Me)
		r.Patch("/me", customerHandler.UpdateMe)
		r.Get("/foto", customerHandler.Foto)
		r.Put("/foto", customerHandler.UpdateFoto)
		r.Get("/upline", customerHandler.Upline)
		r.Get("/haramain/{id}/ulasan", haramainHandler.UlasanSayaAkun)
		r.Put("/haramain/{id}/ulasan", haramainHandler.SimpanUlasanAkun)
		r.Delete("/haramain/{id}/ulasan", haramainHandler.HapusUlasanAkun)
		r.Post("/haramain/ulasan/{id}/laporkan", haramainHandler.LaporkanAkun)
		r.Get("/upline/foto", customerHandler.UplineFoto)
		r.Post("/password", customerHandler.ChangePassword)
		r.Post("/password/buat", customerHandler.BuatPassword)
		r.Get("/hapus", customerHandler.CekHapusAkun)
		r.Post("/hapus", customerHandler.HapusAkun)
		r.Post("/logout", customerHandler.Logout)
		r.Get("/bank-accounts", customerOrders.BankAccounts)
		r.Get("/travelers", customerOrders.Travelers)
		r.Get("/jamaah", jamaahAkunHandler.Daftar)
		r.Get("/pengumuman", pengumumanHandler.Akun)
		r.Get("/notifikasi", notifikasiHandler.Daftar)
		r.Get("/notifikasi/jumlah", notifikasiHandler.Jumlah)
		r.Post("/notifikasi/baca-semua", notifikasiHandler.BacaSemua)
		r.Post("/notifikasi/{id}/baca", notifikasiHandler.Baca)
		r.Get("/notifikasi/pengumuman/{id}", notifikasiHandler.Pengumuman)
		r.Get("/tabungan", tabunganHandler.Ambil)
		r.Put("/tabungan/target", tabunganHandler.SimpanTarget)
		r.Delete("/tabungan/target", tabunganHandler.HapusTarget)
		r.Post("/tabungan/catatan", tabunganHandler.TambahCatatan)
		r.Delete("/tabungan/catatan/{id}", tabunganHandler.HapusCatatan)
		r.Put("/perangkat-push", tabunganHandler.SimpanPerangkat)
		r.Delete("/perangkat-push", tabunganHandler.HapusPerangkat)
		r.Get("/jamaah/{id}", jamaahAkunHandler.Detail)
		r.Put("/jamaah/{id}", jamaahAkunHandler.Simpan)
		r.Get("/bookings", customerOrders.List)
		r.Post("/bookings", selfBookingHandler.CreateBooking)
		r.Post("/bookings-app", selfBookingHandler.CreateBookingApp)
		r.Get("/bookings/{id}", customerOrders.Detail)
		r.Post("/bookings/{id}/cancel", customerOrders.Cancel)
		r.Get("/bookings/{id}/dokumen", customerOrders.Dokumen)
		r.Post("/bookings/{id}/dokumen", customerOrders.KirimDokumen)
		r.Get("/bookings/{id}/progress", customerOrders.Progress)
		r.Get("/bookings/{id}/jamaah", customerOrders.DataJamaah)
		r.Get("/bookings/{id}/surat", customerOrders.SuratDaftar)
		r.Get("/bookings/{id}/surat/{jenis}", customerOrders.SuratUnduh)
		r.Put("/bookings/{id}/jamaah/{jamaah_id}", customerOrders.SimpanDataJamaah)
		r.Get("/bookings/{id}/perlengkapan", customerOrders.Perlengkapan)
		r.Post("/bookings/{id}/perlengkapan/ajukan", customerOrders.AjukanPerlengkapan)
		r.Post("/bookings/{id}/perlengkapan/terima", customerOrders.TerimaPerlengkapan)
		r.Get("/bookings/{id}/manasik", manasikHandler.Booking)
		r.Put("/bookings/{id}/manasik/rsvp", manasikHandler.RSVP)
		r.Get("/bookings/{id}/payments", customerOrders.ListPayments)
		r.Post("/bookings/{id}/payments", customerOrders.CreatePayment)
		r.Post("/media/upload", mediaHandler.UploadPortalMedia)
	})
	r.Route("/api/admin", func(r chi.Router) {
		// Guard: tolak semua admin request jika DB tidak tersedia, LALU validasi auth token JWT
		r.Use(requireDB(db))
		r.Use(identity.RequireAuth)

		// Identitas, brand, dan izin dari token yang sudah divalidasi: dipakai CMS (cms.<domain brand>).
		r.Get("/me", identityHandler.Me)
		r.Get("/my-brand", brandHandler.GetMyBrand)

		// Topbar dashboard: pencarian global dan ringkasan pekerjaan menunggu. Cukup login admin;
		// izin per jenis dan batas brand diperiksa di paket topbar (jenis tanpa izin dikembalikan kosong).
		r.Get("/cari", topbarHandler.Cari)
		r.Get("/notifikasi", topbarHandler.Notifikasi)
		r.Put("/account/password", adminuserHandler.ChangeOwnPassword)

		// Penjaga route per izin. Izin pengguna berasal dari internal/akses
		// (SPEK-AKSES-PENGGUNA-2026-11-25). Alias penjaga yang sering dipakai:
		masterView := r.With(identity.RequirePermission("masterdata.view"))
		masterEdit := r.With(identity.RequirePermission("masterdata.edit"))
		// Daftar master yang dibaca form paket: admin paket (schedule.edit) tanpa izin master ikut boleh membaca.
		masterRead := r.With(identity.RequireAnyPermission("masterdata.view", "schedule.edit"))

		// Hotels
		masterRead.Get("/hotels", hotelHandler.ListHotels)
		masterRead.Get("/hotels/cities", hotelHandler.ListCities)
		masterEdit.Post("/hotels", hotelHandler.CreateHotel)
		masterEdit.Put("/hotels/{id}", hotelHandler.UpdateHotel)
		masterEdit.Delete("/hotels/{id}", hotelHandler.DeleteHotel)

		// Airlines
		masterRead.Get("/airlines", airlineHandler.ListAirlines)
		masterEdit.Post("/airlines", airlineHandler.CreateAirline)
		masterEdit.Put("/airlines/{id}", airlineHandler.UpdateAirline)
		masterEdit.Delete("/airlines/{id}", airlineHandler.DeleteAirline)
		// Master highlight benefit paket (per brand).
		masterView.Get("/benefits", benefitHandler.ListBenefits)
		masterView.Get("/benefits/ikon", benefitHandler.ListIkon)
		masterEdit.Post("/benefits", benefitHandler.CreateBenefit)
		masterEdit.Put("/benefits/{id}", benefitHandler.UpdateBenefit)
		masterEdit.Delete("/benefits/{id}", benefitHandler.DeleteBenefit)

		// Airports
		masterRead.Get("/airports", airportHandler.ListAirports)
		masterEdit.Post("/airports", airportHandler.CreateAirport)
		masterEdit.Put("/airports/{id}", airportHandler.UpdateAirport)
		masterEdit.Delete("/airports/{id}", airportHandler.DeleteAirport)

		// Categories
		masterRead.Get("/categories", categoryHandler.ListCategories)
		masterRead.Get("/categories/{id}", categoryHandler.GetCategory)
		masterEdit.Post("/categories", categoryHandler.CreateCategory)
		masterEdit.Put("/categories/{id}", categoryHandler.UpdateCategory)
		masterEdit.Delete("/categories/{id}", categoryHandler.DeleteCategory)
		r.With(identity.RequirePermission("perlengkapan.view")).Get("/categories/{id}/perlengkapan", perlengkapanHandler.GetCategorySet)
		masterEdit.Put("/categories/{id}/perlengkapan", perlengkapanHandler.PutCategorySet)

		r.Route("/brands", func(r chi.Router) {
			// Daftar ringkas untuk dropdown dan label: cukup login, tanpa brand.view (audit menu Paket T-02).
			r.Get("/options", brandHandler.ListBrandOptions)
			r.With(identity.RequirePermission("brand.view")).Get("/", brandHandler.ListBrands)
			r.With(identity.RequirePermission("brand.view")).Get("/{id}", brandHandler.GetBrand)
			r.With(identity.RequirePermission("brand.view")).Get("/{id}/surat", suratHandler.Get)
			r.With(identity.RequirePermission("brand.edit")).Put("/{id}/surat", suratHandler.Put)
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

		// Pengguna: profil, jenis akses, dan izin (user.admin, hanya Super Admin).
		r.Route("/users", func(r chi.Router) {
			r.Use(identity.RequireHoldingPermission("user.admin"))
			r.Get("/", adminuserHandler.ListUsers)
			r.Post("/", adminuserHandler.CreateUser)
			r.Get("/{id}", adminuserHandler.GetUser)
			r.Put("/{id}", adminuserHandler.UpdateUser)
			r.Put("/{id}/status", adminuserHandler.SetStatus)
			r.Put("/{id}/password", adminuserHandler.ResetPassword)
			r.Delete("/{id}", adminuserHandler.DeleteUser)
		})

		// Media
		r.Get("/media/{category}/{filename}", mediaHandler.ServeProtected)
		r.Post("/media/upload", mediaHandler.UploadMedia)

		// Itineraries
		masterRead.Get("/itineraries", itineraryHandler.ListItineraries)
		masterRead.Get("/itineraries/{id}", itineraryHandler.GetItinerary)
		masterEdit.Post("/itineraries", itineraryHandler.CreateItinerary)
		masterEdit.Put("/itineraries/{id}", itineraryHandler.UpdateItinerary)
		masterEdit.Delete("/itineraries/{id}", itineraryHandler.DeleteItinerary)

		// Schedules
		scheduleView := r.With(identity.RequirePermission("schedule.view"))
		scheduleEdit := r.With(identity.RequirePermission("schedule.edit"))
		scheduleView.Get("/schedules", scheduleHandler.ListSchedulesAdmin)
		// Pilihan benefit untuk form paket: admin paket boleh memilih dan membuat benefit baru tanpa izin
		// master, tetapi tidak mengubah atau menonaktifkan yang sudah ada. Didaftarkan sebelum /schedules/{id};
		// chi mendahulukan segmen statis, jadi "benefit-options" tidak terbaca sebagai id.
		scheduleView.Get("/schedules/benefit-options", benefitHandler.ListOpsi)
		scheduleEdit.Post("/schedules/benefit-options", benefitHandler.CreateOpsi)
		// Lokasi Dikunjungi: active Jelajah Haramain places; static segment, registered before /schedules/{id}.
		scheduleView.Get("/schedules/lokasi-options", scheduleHandler.ListLokasiOptions)
		scheduleView.Get("/schedules/{id}", scheduleHandler.GetScheduleAdmin)
		scheduleView.Get("/schedules/{id}/seat", scheduleHandler.GetScheduleSeat)
		scheduleEdit.Post("/schedules", scheduleHandler.CreateSchedule)
		scheduleEdit.Put("/schedules/{id}", scheduleHandler.UpdateSchedule)
		scheduleEdit.Put("/schedules/{id}/status", scheduleHandler.UpdateScheduleStatus)
		scheduleEdit.Post("/schedules/{id}/seat/samakan", scheduleHandler.SamakanScheduleSeat)
		scheduleEdit.Put("/schedules/{id}/seat-tampil", scheduleHandler.UpdateScheduleSeatTampil)
		scheduleEdit.Delete("/schedules/{id}", scheduleHandler.DeleteSchedule)
		r.With(identity.RequirePermission("perlengkapan.view")).Get("/schedules/{id}/perlengkapan", perlengkapanHandler.GetScheduleSet)
		scheduleEdit.Put("/schedules/{id}/perlengkapan", perlengkapanHandler.PutScheduleSet)
		scheduleView.Get("/schedules/{id}/manasik", manasikHandler.UntukPaket)

		// Manifest paket: PNR, pemeriksaan data pax, unduh .xlsx (SPEK-MANIFEST-PAKET-2026-11-25).
		manifestKelola := r.With(identity.RequirePermission("manifest.manage"))
		manifestKelola.Get("/schedules/{id}/manifest", manifestHandler.Detail)
		manifestKelola.Put("/schedules/{id}/manifest", manifestHandler.Simpan)
		manifestKelola.Get("/schedules/{id}/manifest/unduh", manifestHandler.Unduh)

		// Manasik: satu manasik untuk satu atau lebih paket, rekap kehadiran, dan absensi hari-H.
		manasikView := r.With(identity.RequirePermission("manasik.view"))
		manasikEdit := r.With(identity.RequirePermission("manasik.edit"))
		manasikView.Get("/manasik", manasikHandler.Daftar)
		manasikEdit.Get("/manasik/paket-tersedia", manasikHandler.PaketTersedia)
		manasikEdit.Post("/manasik", manasikHandler.Buat)
		manasikView.Get("/manasik/{id}", manasikHandler.Detail)
		manasikEdit.Put("/manasik/{id}", manasikHandler.Ubah)
		manasikEdit.Delete("/manasik/{id}", manasikHandler.Hapus)
		manasikEdit.Post("/manasik/{id}/paket", manasikHandler.TambahPaket)
		manasikEdit.Delete("/manasik/{id}/paket/{schedule_id}", manasikHandler.LepasPaket)
		manasikEdit.Put("/manasik/{id}/absensi", manasikHandler.Absensi)

		// Jamaah
		jamaahView := r.With(identity.RequirePermission("jamaah.view"))
		jamaahEdit := r.With(identity.RequirePermission("jamaah.edit"))
		jamaahView.Get("/jamaah", jamaahHandler.ListJamaah)
		jamaahView.Get("/jamaah/{id}", jamaahHandler.GetJamaah)
		jamaahEdit.Post("/jamaah", jamaahHandler.CreateJamaah)
		jamaahEdit.Put("/jamaah/{id}", jamaahHandler.UpdateJamaah)
		jamaahEdit.Put("/jamaah/{id}/catatan", jamaahHandler.UpdateCatatan)
		jamaahEdit.Put("/jamaah/{id}/tanggal-lahir", jamaahHandler.UpdateTanggalLahir)
		jamaahEdit.Delete("/jamaah/{id}", jamaahHandler.DeleteJamaah)

		// Dokumen Jamaah
		r.With(identity.RequirePermission("dokumen.view")).Get("/jamaah/{jamaah_id}/dokumen", dokumenHandler.ListDokumen)
		r.With(identity.RequirePermission("dokumen.edit")).Post("/jamaah/{jamaah_id}/dokumen", dokumenHandler.UpsertDokumen)
		r.With(identity.RequirePermission("dokumen.edit")).Put("/dokumen/{id}/status", dokumenHandler.UpdateDokumenStatus)
		r.With(identity.RequirePermission("dokumen.view")).Get("/dokumen/{id}/riwayat", dokumenHandler.RiwayatDokumen)

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
		bookingEdit.Post("/bookings/{id}/discounts", bookingHandler.AddBookingDiscount)
		bookingEdit.Delete("/bookings/{id}/discounts/{discountID}", bookingHandler.RemoveBookingDiscount)
		bookingView.Get("/bookings/{id}/cashback", bookingHandler.GetKreditCashback)
		bookingView.Get("/bookings/{id}/agen-komisi", bookingHandler.GetAgenKomisi)
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
		r.With(identity.RequirePermission("perlengkapan.view")).Get("/perlengkapan-mutasi", perlengkapanHandler.ListMutasi)
		r.With(identity.RequirePermission("perlengkapan.view")).Get("/perlengkapan-pengajuan", perlengkapanHandler.ListPengajuan)
		r.With(identity.RequirePermission("perlengkapan.edit")).Post("/perlengkapan-mutasi/pembelian", perlengkapanHandler.Pembelian)
		r.With(identity.RequirePermission("perlengkapan.edit")).Post("/perlengkapan-mutasi/koreksi", perlengkapanHandler.Koreksi)

		// Perlengkapan Set Template (Global) — master data holding.
		r.With(identity.RequirePermission("perlengkapan.view")).Get("/perlengkapan-set-template", perlengkapanHandler.GetSetTemplate)
		masterEdit.Put("/perlengkapan-set-template", perlengkapanHandler.UpdateSetTemplate)

		// Payments
		r.With(identity.RequirePermission("payment.view")).Get("/payments", paymentHandler.ListAllPayments)
		r.With(identity.RequirePermission("payment.view")).Get("/bookings/{booking_id}/payments", paymentHandler.ListPayments)
		r.With(identity.RequirePermission("payment.edit")).Post("/bookings/{booking_id}/payments", paymentHandler.CreatePayment)
		r.With(identity.RequirePermission("payment.verify")).Put("/payments/{id}/status", paymentHandler.UpdatePaymentStatus)
		r.With(identity.RequirePermission("payment.verify")).Delete("/payments/{id}", paymentHandler.DeletePayment)
		r.With(identity.RequirePermission("payment.void")).Post("/payments/{id}/void", paymentHandler.VoidPayment)

		// Akun Pelanggan: daftar dan detail akun yang bisa booking (baca saja; brand scope dicek di query).
		r.Route("/akun", func(r chi.Router) {
			r.Use(identity.RequirePermission("akun.view"))
			r.Get("/", adminAkunHandler.List)
			r.Get("/agen-dari-tautan", adminAkunHandler.AgenDariTautan)
			r.Get("/{id}", adminAkunHandler.Detail)
			r.With(identity.RequirePermission("akun.edit")).Post("/{id}/lepas-tautan", adminAkunHandler.LepasTautan)
			r.With(identity.RequirePermission("akun.edit")).Post("/aktivasi/reset-hitung", aktivasiHandler.ResetHitung)
			// Izin menurut status akun diperiksa di handler (sama seperti ubah NIK agen).
			r.Post("/{id}/kosongkan-nik-agen", adminAkunHandler.KosongkanNIKAgen)
			r.With(identity.RequirePermission("akun.edit")).Put("/{id}", adminAkunHandler.UbahProfil)
			// Izin NIK agen (agen.approve atau agen.kaitan.edit) diperiksa di handler menurut status akun.
			r.Put("/{id}/nik-agen", adminAkunHandler.UbahNIK)
		})

		// Pengumuman untuk pelanggan (tampil di dashboard akun web).
		r.Route("/banner-promosi", func(r chi.Router) {
			r.Use(identity.RequirePermission("banner.manage"))
			r.Get("/", bannerHandler.Daftar)
			r.Post("/", bannerHandler.Buat)
			r.Put("/urutan", bannerHandler.Urutkan)
			r.Put("/{id}", bannerHandler.Ubah)
			r.Delete("/{id}", bannerHandler.Hapus)
		})
		r.Route("/haramain", func(r chi.Router) {
			r.Use(identity.RequirePermission("haramain.manage"))
			r.Get("/", haramainHandler.DaftarAdmin)
			r.Post("/", haramainHandler.BuatAdmin)
			r.Put("/urutan", haramainHandler.UrutkanAdmin)
			r.Get("/{id}", haramainHandler.DetailAdmin)
			r.Put("/{id}", haramainHandler.UbahAdmin)
			r.Delete("/{id}", haramainHandler.HapusAdmin)
		})
		r.Route("/haramain-ulasan", func(r chi.Router) {
			r.Use(identity.RequirePermission("haramain.manage"))
			r.Get("/", haramainHandler.UlasanAdminList)
			r.Put("/{id}/tampil", haramainHandler.AturTampilAdmin)
			r.Post("/{id}/abaikan", haramainHandler.AbaikanAdmin)
		})
		r.Route("/pengumuman", func(r chi.Router) {
			r.Use(identity.RequirePermission("pengumuman.manage"))
			r.Get("/", pengumumanHandler.DaftarAdmin)
			r.Post("/", pengumumanHandler.Buat)
			r.Put("/{id}", pengumumanHandler.Ubah)
			r.Delete("/{id}", pengumumanHandler.Hapus)
		})

		// Persetujuan perubahan data jamaah dari pelanggan (halaman Data Jamaah di akun).
		r.Route("/perubahan-jamaah", func(r chi.Router) {
			r.Use(identity.RequirePermission("jamaah.perubahan.approve"))
			r.Get("/", jamaahAkunHandler.ListAdmin)
			r.Get("/{id}", jamaahAkunHandler.DetailAdmin)
			r.Post("/{id}/setujui", jamaahAkunHandler.Setujui)
			r.Post("/{id}/tolak", jamaahAkunHandler.Tolak)
		})

		// Agen Syiar — Persetujuan Agen & status (screen B1/B2a); brand scope
		// dicek di repository.
		r.Route("/agen", func(r chi.Router) {
			r.Use(identity.RequirePermission("agen.view"))
			r.Get("/pengajuan", agenHandler.ListPengajuan)
			r.Get("/aktif", agenHandler.ListAgenAktif)
			r.Get("/", agenHandler.ListAgen)
			r.Get("/komisi", agenHandler.RiwayatKomisiAdmin)
			r.Get("/peringkat", agenHandler.PeringkatAgen)
			r.Get("/{akunID}", agenHandler.DetailAgen)
			// Batas hold seat agen (keputusan 12 November 2026): lihat dengan agen.view, ubah dengan agen.edit.
			r.Get("/{akunID}/batas-seat", agenHandler.GetBatasSeat)
			r.With(identity.RequirePermission("agen.edit")).Put("/{akunID}/batas-seat", agenHandler.UbahBatasSeat)
			r.With(identity.RequirePermission("agen.approve")).Post("/{akunID}/setujui", agenHandler.Setujui)
			r.With(identity.RequirePermission("agen.approve")).Post("/{akunID}/tolak", agenHandler.Tolak)
			r.With(identity.RequirePermission("agen.approve")).Post("/{akunID}/keputusan", agenHandler.Keputusan)
			r.With(identity.RequirePermission("agen.edit")).Put("/{akunID}/status", agenHandler.UbahStatus)
			r.With(identity.RequirePermission("agen.approve")).Post("/pembayaran/{id}/verifikasi", agenHandler.VerifikasiPembayaran)
			r.With(identity.RequirePermission("agen.approve")).Post("/pembayaran/{id}/tolak", agenHandler.TolakPembayaran)
			r.With(identity.RequirePermission("agen.approve")).Post("/pembayaran/{id}/bukti", agenHandler.UploadBuktiAdmin)
		})

		// Agen Syiar — Persetujuan Pencairan (C1), lintas brand. Izin agen.payout.approve
		// hanya bisa diberikan ke pengguna Holding (akses.ValidasiIzinTim).
		r.Group(func(r chi.Router) {
			r.Use(identity.RequirePermission("agen.payout.approve"))
			r.Get("/pencairan", agenHandler.ListPencairanAdmin)
			r.Get("/pencairan/ringkasan", agenHandler.RingkasanPencairan)
			r.Post("/pencairan/{id}/setujui", agenHandler.SetujuiPencairan)
			r.Post("/pencairan/{id}/tolak", agenHandler.TolakPencairan)
		})
		r.Group(func(r chi.Router) {
			r.Use(identity.RequirePermission("agen.kaitan.edit"))
			r.Get("/kaitan-agen", agenHandler.CariJamaahKaitan)
			r.Get("/kaitan-agen/{jamaahID}/log", agenHandler.LogKaitan)
			r.Put("/kaitan-agen/{jamaahID}", agenHandler.GantiKaitan)
			r.Get("/kaitan-agen/akun", agenHandler.CariAkunKaitan)
			r.Get("/kaitan-agen/akun/{akunID}/log", agenHandler.LogKaitanAkun)
			r.Get("/kaitan-agen/akun/{akunID}/jamaah-ikut", agenHandler.JamaahIkutUpline)
			r.Put("/kaitan-agen/akun/{akunID}", agenHandler.GantiAgenAkun)
		})

		// Analytics lintas brand
		r.With(identity.RequirePermission("report.analytics.view")).Get("/analytics/transactions-30-days", paymentHandler.ListDailyBrandTransactions)
		r.With(identity.RequirePermission("report.analytics.view")).Get("/analytics/pax-30-days", bookingHandler.ListDailyBrandPax)
		r.With(identity.RequirePermission("report.analytics.view")).Get("/analytics/marketing-ringkas", analitikHandler.MarketingRingkas)
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

// corsOrigins: origin pengembangan (localhost, *.test) hanya bila BUKAN produksi; di produksi
// hanya CORS_ALLOWED_ORIGINS.
func corsOrigins(production bool, configured string) []string {
	allowedOrigins := []string{}
	if !production {
		allowedOrigins = append(allowedOrigins,
			"http://localhost:5173",
			"http://localhost:3000", "https://localhost:3000",
			"http://*.azhan.test", "http://*.azhan.test:3000",
			"https://*.azhan.test", "https://*.azhan.test:3000",
			"http://*.test", "https://*.test",
		)
	}
	for _, origin := range strings.Split(configured, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			allowedOrigins = append(allowedOrigins, origin)
		}
	}
	return allowedOrigins
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

// runPelunasanExpiry membatalkan booking dp yang melewati tenggat lunas dan memberi tahu pemiliknya. Hari tenggat masih
// termasuk; pembayaran yang sudah masuk tidak disentuh (urusan keuangan ditangani manual). Dimatikan dengan
// PELUNASAN_AUTO_BATAL=false.
func runPelunasanExpiry(repo *booking.Repository, db *sql.DB) {
	jalan := func() {
		ids, err := repo.BatalkanLewatBatasPelunasan(context.Background(), time.Now())
		if err != nil {
			log.Printf("[WARN] Gagal membatalkan booking lewat batas pelunasan: %v", err)
			return
		}
		for _, id := range ids {
			notifikasi.BookingBatalJatuhTempo(context.Background(), db, id)
		}
		if len(ids) > 0 {
			log.Printf("[INFO] Booking dibatalkan karena lewat batas pelunasan: %v", ids)
		}
	}
	jalan()
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		jalan()
	}
}

func runSeatHoldExpiry(repo *booking.Repository) {
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

// runAktivasiCleanup membuang penghitung percobaan dan log aktivasi akun jamaah yang sudah lewat masa simpan.
// Jalan sekali saat start lalu tiap 6 jam, juga saat flag aktivasi mati (tabel tetap bisa terisi dari aksi admin).
func runAktivasiCleanup(db *sql.DB) {
	jalan := func() {
		h, err := customer.BersihkanAktivasi(context.Background(), db)
		if err != nil {
			log.Printf("[WARN] Gagal membersihkan data aktivasi akun jamaah: %v", err)
			return
		}
		if h.Hitung+h.LogGagal+h.LogJejak > 0 {
			log.Printf("[INFO] Pembersihan aktivasi akun jamaah: penghitung=%d log_gagal=%d log_jejak=%d", h.Hitung, h.LogGagal, h.LogJejak)
		}
	}
	jalan()
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		jalan()
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
