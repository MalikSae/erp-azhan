// tautanAman: URL dari data (video hotel, brosur paket) hanya dipakai sebagai href bila berupa tautan web
// biasa atau path di server ini. Skema lain, misalnya javascript:, diganti '#'.
export function tautanAman(url) {
  const nilai = String(url || '').trim();
  if (/^https?:\/\//i.test(nilai) || (nilai.startsWith('/') && !nilai.startsWith('//'))) return nilai;
  return '#';
}
