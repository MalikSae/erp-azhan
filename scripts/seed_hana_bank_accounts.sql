-- Local UI test fixtures only: dummy bank accounts for Hana (brand 2).
-- Account numbers are synthetic, NOT Hana's real accounts; replace via ERP before go-live.
-- Marked with instructions 'DATA TESTING%'. Idempotent: only inserts missing (brand, number) pairs.
SET NAMES utf8mb4;
START TRANSACTION;
SET @hana = (SELECT id FROM brands WHERE domain='hana.azhan.test' AND id=2);
INSERT INTO bank_accounts (brand_id, bank_name, account_number, account_holder, instructions, is_active, sort_order)
SELECT @hana, n.bank, n.nomor, 'PT Hana Asia Wisata', 'DATA TESTING: rekening simulasi, bukan rekening asli.', 1, n.urut
FROM (
 SELECT 'BSI' bank, '7100000001' nomor, 1 urut
 UNION ALL SELECT 'Mandiri', '1370000000001', 2
 UNION ALL SELECT 'BCA', '0000000001', 3
) n WHERE @hana IS NOT NULL AND NOT EXISTS (
 SELECT 1 FROM bank_accounts b WHERE b.brand_id=@hana AND b.account_number=n.nomor
);
SELECT ROW_COUNT() AS inserted_bank_accounts;
COMMIT;
SELECT id,bank_name,account_number,account_holder,is_active,sort_order
FROM bank_accounts WHERE brand_id=@hana AND instructions LIKE 'DATA TESTING%' ORDER BY sort_order;
