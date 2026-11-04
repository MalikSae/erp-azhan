-- 117_akun_view.sql
-- Izin baru untuk layar Akun Pelanggan (daftar dan detail akun yang bisa booking).
-- Peran yang sudah boleh melihat jamaah atau agen ikut boleh melihat akun; tidak ada peran baru yang mendapat akses data.

INSERT IGNORE INTO permissions (code, description) VALUES
  ('akun.view', 'Lihat daftar dan detail akun pelanggan');

INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT DISTINCT rp.role_id, baru.id
FROM role_permissions rp
JOIN permissions lama ON lama.id = rp.permission_id AND lama.code IN ('jamaah.view', 'agen.view')
JOIN permissions baru ON baru.code = 'akun.view';
