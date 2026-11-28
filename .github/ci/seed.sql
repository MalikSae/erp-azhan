-- Data minimal untuk integration test di CI (ERP_TEST_DB=1). Dijalankan setelah semua migrasi.
-- Fixture internal/testdb memakai jadwal pertama yang brand-nya punya kode_brand.

UPDATE brands SET kode_brand = 'HN', domain = 'hana.test' WHERE id = 1;

INSERT INTO schedules (brand_id, jadwal_nama, berangkat_tanggal, pulang_tanggal,
  harga_quad, harga_triple, harga_double, seat_total, seat_sisa)
VALUES (1, 'Paket Uji CI', DATE_ADD(CURDATE(), INTERVAL 90 DAY), DATE_ADD(CURDATE(), INTERVAL 99 DAY),
  30000000, 32000000, 35000000, 45, 45);

INSERT INTO brands (name, kode_brand, domain) VALUES ('Brand Uji Kedua', 'AL', 'alsha.test');

INSERT INTO schedules (brand_id, jadwal_nama, berangkat_tanggal, pulang_tanggal,
  harga_quad, harga_triple, harga_double, seat_total, seat_sisa)
VALUES
  (1, 'Paket Uji CI Kedua', DATE_ADD(CURDATE(), INTERVAL 120 DAY), DATE_ADD(CURDATE(), INTERVAL 129 DAY),
   30000000, 32000000, 35000000, 45, 45),
  (2, 'Paket Uji CI Brand Kedua', DATE_ADD(CURDATE(), INTERVAL 90 DAY), DATE_ADD(CURDATE(), INTERVAL 99 DAY),
   30000000, 32000000, 35000000, 45, 45);

-- Hash bcrypt acak; akun ini tidak dipakai untuk login.
INSERT INTO admin_users (email, display_name, password_hash, brand_id, is_super_admin, is_active)
VALUES ('super-ci@uji.test', 'Super Admin CI', '$2a$10$abcdefghijklmnopqrstuuJ0Ck1yO5o3s0f3m0d8b6Qk0lHn3rGe', NULL, TRUE, TRUE);
