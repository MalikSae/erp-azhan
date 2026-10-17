-- NIK jamaah tersimpan di akun pelanggan (isi otomatis di checkout berikutnya).
ALTER TABLE customer_travelers
    ADD COLUMN nik VARCHAR(20) NULL AFTER no_hp;
