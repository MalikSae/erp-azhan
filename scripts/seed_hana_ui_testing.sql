-- Local UI test fixtures only. Source: https://hanahajiumroh.com/ (2026-10-01).
-- Source values: names, departure dates, starting prices, hotels, benefits.
-- Synthetic: return dates, room upgrades, seats, promo, and two duration variants.
-- Idempotent: only inserts missing fixture schedules (by name) for Hana; no updates/deletes.
SET NAMES utf8mb4;
START TRANSACTION;
SET @hana = (SELECT id FROM brands WHERE domain='hana.azhan.test' AND id=2);
INSERT INTO hotels (name, city, star_rating, distance_m)
SELECT n.name, IF(n.name IN ('Royal Andalus','Andalus As-Salam','Shaza Regency'),'Madinah','Makkah'), NULL, NULL FROM (
 SELECT 'Azka Shafa' name UNION ALL SELECT 'Royal Andalus' UNION ALL SELECT 'Olayan Golden'
 UNION ALL SELECT 'Andalus As-Salam' UNION ALL SELECT 'Winner Inn'
 UNION ALL SELECT 'Sofwah Tower' UNION ALL SELECT 'Shaza Regency'
) n WHERE @hana IS NOT NULL AND NOT EXISTS (SELECT 1 FROM hotels h WHERE h.name=n.name);
INSERT INTO schedules (
 brand_id,jadwal_nama,status,is_promo,is_direct_flight,seat_total,seat_sisa,
 maskapai_id,berangkat_tanggal,pulang_tanggal,hotel_mekkah_id,hotel_madinah_id,
 harga_quad,harga_triple,harga_double,harga_coret,promo_until,include_items,exclude_items,
 nominal_komisi_langsung,nominal_bonus_pembinaan
)
SELECT @hana,n.name,'published',n.promo,1,45,n.remaining,
 (SELECT id FROM airlines WHERE name='SAUDIA' LIMIT 1),n.depart,DATE_ADD(n.depart,INTERVAL n.nights DAY),
 (SELECT id FROM hotels WHERE name=n.mekkah LIMIT 1),(SELECT id FROM hotels WHERE name=n.madinah LIMIT 1),
 n.price,n.price+2000000,n.price+4000000,IF(n.promo,n.price+1000000,NULL),IF(n.promo,'2026-10-20',NULL),
 JSON_ARRAY('Manasik','Albaik','City Tour Thaif','Sertifikat Umroh'),
 JSON_ARRAY('DATA TESTING: tanggal pulang, harga kamar, kuota, dan promo merupakan simulasi.'),0,0
FROM (
 SELECT 'Umroh Milenial Combine Bintang 5' name,DATE('2026-10-20') depart,8 nights,30999000 price,0 promo,45 remaining,'Azka Shafa' mekkah,'Royal Andalus' madinah
 UNION ALL SELECT 'Umroh Milenial Oktober','2026-10-29',8,29999000,1,8,'Olayan Golden','Andalus As-Salam'
 UNION ALL SELECT 'Umroh Milenial Combine November','2026-11-11',8,32999000,0,32,'Azka Shafa','Royal Andalus'
 UNION ALL SELECT 'Umroh Milenial November','2026-11-18',8,29999000,0,40,'Olayan Golden','Andalus As-Salam'
 UNION ALL SELECT 'Umroh Milenial Desember','2026-12-02',8,29199000,0,20,'Winner Inn','Andalus As-Salam'
 UNION ALL SELECT 'Umroh Super Milenial','2026-12-26',8,44999000,0,12,'Sofwah Tower','Shaza Regency'
 UNION ALL SELECT 'Simulasi Milenial 12 Hari','2026-11-23',11,34999000,0,45,'Azka Shafa','Royal Andalus'
 UNION ALL SELECT 'Simulasi Super Milenial 15 Hari','2026-12-21',14,46999000,0,35,'Sofwah Tower','Shaza Regency'
) n WHERE @hana IS NOT NULL AND NOT EXISTS (SELECT 1 FROM schedules s WHERE s.brand_id=@hana AND s.jadwal_nama=n.name);
SELECT ROW_COUNT() AS inserted_schedules;
COMMIT;
SELECT id,jadwal_nama,berangkat_tanggal,pulang_tanggal,harga_quad,seat_total,seat_sisa,status
FROM schedules WHERE brand_id=@hana AND JSON_SEARCH(exclude_items, 'one', 'DATA TESTING%') IS NOT NULL ORDER BY berangkat_tanggal;


