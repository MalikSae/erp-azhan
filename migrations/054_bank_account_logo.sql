-- Tambah kolom logo_url pada tabel bank_accounts
ALTER TABLE bank_accounts ADD COLUMN logo_url VARCHAR(500) NULL AFTER bank_name;
