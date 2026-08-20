-- Samakan jenis dokumen portal dengan daftar dokumen jamaah yang didukung aplikasi.
ALTER TABLE dokumen_jamaah
  MODIFY COLUMN jenis ENUM(
    'pas_foto',
    'paspor',
    'ktp',
    'kk',
    'buku_nikah',
    'akte_lahir',
    'vaksin_meningitis'
  ) NOT NULL;
