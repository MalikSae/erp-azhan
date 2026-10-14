-- 084_schedule_seat_adjustments.sql
-- Riwayat penyesuaian sisa kursi manual (PUT /api/admin/schedules/{id}/seat),
-- audit manajemen seat SM-03. Sebelumnya hanya tercatat di log server tanpa
-- pelaku. admin_user_id NULL bila akun pelaku kemudian dihapus; riwayat ikut
-- terhapus bila paketnya dihapus (paket yang punya booking tidak bisa dihapus).
CREATE TABLE IF NOT EXISTS schedule_seat_adjustments (
  id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  schedule_id    BIGINT UNSIGNED NOT NULL,
  admin_user_id  BIGINT UNSIGNED NULL,
  seat_before    INT NOT NULL,
  seat_after     INT NOT NULL,
  reason         VARCHAR(500) NOT NULL,
  created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_seat_adj_schedule (schedule_id, created_at),
  CONSTRAINT fk_seat_adj_schedule FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE,
  CONSTRAINT fk_seat_adj_admin FOREIGN KEY (admin_user_id) REFERENCES admin_users(id) ON DELETE SET NULL
) ENGINE=InnoDB;
