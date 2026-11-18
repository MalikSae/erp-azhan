import test from 'node:test';
import assert from 'node:assert/strict';
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

// Penjaga copy dashboard (docs/ux-copy.md): frasa terlarang tidak boleh masuk lagi ke teks tampil.
// Jalankan: npm run test:copy
export const FORBIDDEN = [
  [/\bPassword\b/, 'Pakai "kata sandi"'],
  [/\bpassword (baru|saat ini|lama|minimal|maksimal)\b/, 'Pakai "kata sandi"'],
  [/Yakin ingin|Apakah Anda yakin/, 'Dialog: sebut aksi dan akibat, bukan "Yakin?"'],
  [/Tindakan ini tidak (bisa|dapat) dibatalkan/, 'Tanpa kalimat penjelas; pakai "Data tidak bisa dikembalikan."'],
  [/placeholder=(?:"|\{['"`])Contoh:/, 'Placeholder berisi contoh nyata, tanpa "Contoh:"'],
  [/placeholder="(Masukkan|Ketik|Ulangi) /, 'Placeholder tanpa kata kerja; kosongkan bila tidak menambah informasi'],
  [/\bSilakan\b/i, 'Tanpa "silakan"; langsung kata kerja ("Coba lagi.")'],
  [/Terjadi kesalahan/i, 'Sebut masalahnya: "Data gagal disimpan. Coba lagi."'],
  [/\bmaaf\b/i, 'Tanpa permintaan maaf'],
  [/berhasil (dibuat|diubah|diperbarui|ditambahkan|dihapus|disimpan|diganti)/i, 'Pola: "Perubahan disimpan." / "Booking dibuat."'],
  [/[A-Za-z]!(?=['"`<])/, 'Tanpa tanda seru di pesan sistem'],
  [/\b(kamu|peserta|penumpang|kursi|pesanan|pemesanan|reservasi|blokir|cicilan|uang muka)\b/i, 'Ikuti istilah di GLOSSARY.md'],
];

export function stripComments(src) {
  return src
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/(^|[^:'"`])\/\/.*$/gm, '$1');
}

export function findViolations(src) {
  const out = [];
  stripComments(src).split('\n').forEach((line, i) => {
    for (const [re, hint] of FORBIDDEN) {
      const m = re.exec(line);
      if (m) out.push({ line: i + 1, found: m[0], hint });
    }
  });
  return out;
}

const SRC = fileURLToPath(new URL('../', import.meta.url));

function walk(dir, acc = []) {
  for (const e of readdirSync(dir, { withFileTypes: true })) {
    const p = join(dir, e.name);
    if (e.isDirectory()) walk(p, acc);
    else if (/\.(jsx?|mjs)$/.test(e.name) && !/\.test\.js$/.test(e.name)) acc.push(p);
  }
  return acc;
}

test('penjaga mendeteksi frasa terlarang', () => {
  assert.equal(findViolations('<h3>Ubah password</h3>').length, 0, 'huruf kecil tunggal dibiarkan; kapital dan frasa yang dijaga');
  assert.equal(findViolations("label='Password baru'").length >= 1, true);
  assert.equal(findViolations("'Yakin ingin menghapus X?'").length, 1);
  assert.equal(findViolations('placeholder="Contoh: BCA"').length, 1);
  assert.equal(findViolations("setError('Silakan coba lagi.')").length, 1);
  assert.equal(findViolations("name='password' type='password'").length, 0);
  assert.equal(findViolations('// Silakan abaikan').length, 0);
});

test('teks tampil dashboard mengikuti docs/ux-copy.md', () => {
  const found = [];
  for (const file of walk(SRC)) {
    for (const v of findViolations(readFileSync(file, 'utf8'))) {
      found.push(`${file.slice(SRC.length)}:${v.line} "${v.found}" — ${v.hint}`);
    }
  }
  assert.deepEqual(found, [], `\n${found.join('\n')}\n`);
});
