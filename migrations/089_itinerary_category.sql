-- 089_itinerary_category.sql
-- Itinerary dikaitkan ke kategori paket (opsional) agar pilihan itinerary di form
-- paket bisa disaring menurut kategori. Kategori yang dihapus -> itinerary menjadi
-- tanpa kategori (SET NULL), bukan ikut terhapus.
ALTER TABLE itineraries
  ADD COLUMN category_id BIGINT UNSIGNED NULL AFTER title,
  ADD CONSTRAINT fk_itineraries_category FOREIGN KEY (category_id)
    REFERENCES package_categories(id) ON DELETE SET NULL;

-- Isi awal dari judul (mis. "Umroh Plus Dubai 9 Hari" -> kategori "Plus").
UPDATE itineraries i
JOIN package_categories pc ON LOCATE(pc.name, i.title) > 0
SET i.category_id = pc.id
WHERE i.category_id IS NULL;
