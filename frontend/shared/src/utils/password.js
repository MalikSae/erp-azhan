// Batas panjang password di server: 72 byte (batas bcrypt). Huruf beraksen dan emoji memakai lebih dari
// satu byte, jadi yang dihitung byte, bukan jumlah karakter.
const MAKS_BYTE_PASSWORD = 72;

export function passwordTerlaluPanjang(password) {
  return new TextEncoder().encode(password || '').length > MAKS_BYTE_PASSWORD;
}
