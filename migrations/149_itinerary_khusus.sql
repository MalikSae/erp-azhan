-- 149: Itinerary khusus per paket (keputusan 17 November 2026).
--
-- Paket = satu baris schedules. Itinerary khusus = salinan pribadi milik satu paket, dimulai dari master lalu
-- boleh berbeda. Baris master punya schedule_id NULL; baris pribadi menyimpan id paket pemiliknya. Tanpa FK ke
-- schedules supaya tidak melingkar dengan schedules.itinerary_id; aplikasi menghapus baris pribadi sendiri.
ALTER TABLE itineraries
  ADD COLUMN schedule_id BIGINT UNSIGNED NULL AFTER category_id,
  ADD UNIQUE KEY uq_itineraries_schedule (schedule_id);
