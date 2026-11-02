import React, { useEffect, useState } from 'react';
import Button from 'shared/ui/Button';
import Input from 'shared/ui/Input';
import Alert from 'shared/ui/Alert';
import FileInput from 'shared/ui/FileInput';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import { getBrandSurat, updateBrandSurat } from '../api/brandSurat';
import { uploadMedia } from '../api/media';

const kosong = {
  nama_perusahaan: '',
  kode_surat: '',
  nomor_terakhir: 0,
  penandatangan_nama: '',
  penandatangan_jabatan: '',
  kop_url: '',
  tanda_tangan_url: '',
  stempel_url: '',
};

const GAMBAR = [
  { kunci: 'kop_url', label: 'Kop Surat', petunjuk: 'Gambar kop selebar halaman (logo, nama, alamat). Kosongkan untuk memakai kop teks dari data brand.' },
  { kunci: 'tanda_tangan_url', label: 'Tanda Tangan', petunjuk: 'PNG dengan latar transparan.' },
  { kunci: 'stempel_url', label: 'Stempel', petunjuk: 'PNG dengan latar transparan.' },
];

// Isian surat jamaah per brand (Surat Keterangan Umroh dan Surat Rekomendasi Paspor yang diunduh jamaah dari halaman booking).
// Kota surat dan nomor izin PPIU/PIHK tidak diisi di sini: diambil dari bagian Kontak & Alamat dan Legalitas.
const BrandSurat = ({ brandId }) => {
  const [form, setForm] = useState(kosong);
  const [awal, setAwal] = useState(kosong);
  const [memuat, setMemuat] = useState(true);
  const [menyimpan, setMenyimpan] = useState(false);
  const [mengunggah, setMengunggah] = useState('');
  const [pratinjau, setPratinjau] = useState({});
  const [galat, setGalat] = useState('');
  const [berhasil, setBerhasil] = useState(false);

  useEffect(() => {
    let batal = false;
    (async () => {
      try {
        const data = await getBrandSurat(brandId);
        if (batal) return;
        const isian = { ...kosong, ...Object.fromEntries(Object.entries(data).filter(([k]) => k in kosong).map(([k, v]) => [k, v ?? kosong[k]])) };
        setForm(isian);
        setAwal(isian);
      } catch (err) {
        if (!batal) setGalat(err?.response?.data?.error || 'Gagal memuat pengaturan surat.');
      } finally {
        if (!batal) setMemuat(false);
      }
    })();
    return () => { batal = true; };
  }, [brandId]);

  const ubah = (e) => {
    const { name, value } = e.target;
    setBerhasil(false);
    setForm((f) => ({ ...f, [name]: name === 'nomor_terakhir' ? Math.max(0, parseInt(value, 10) || 0) : value }));
  };

  const unggah = (kunci) => async (e) => {
    const file = e.target.files[0];
    if (!file) return;
    setPratinjau((p) => ({ ...p, [kunci]: URL.createObjectURL(file) }));
    setMengunggah(kunci);
    setGalat('');
    setBerhasil(false);
    try {
      const url = await uploadMedia(file, 'brand-logos');
      setForm((f) => ({ ...f, [kunci]: url }));
    } catch {
      setGalat('Gagal mengunggah gambar.');
    } finally {
      setMengunggah('');
    }
  };

  const hapus = (kunci) => () => {
    setPratinjau((p) => ({ ...p, [kunci]: null }));
    setForm((f) => ({ ...f, [kunci]: '' }));
    setBerhasil(false);
  };

  const simpan = async () => {
    setMenyimpan(true);
    setGalat('');
    setBerhasil(false);
    try {
      const hasil = await updateBrandSurat(brandId, form);
      const isian = { ...kosong, ...Object.fromEntries(Object.entries(hasil).filter(([k]) => k in kosong).map(([k, v]) => [k, v ?? kosong[k]])) };
      setForm(isian);
      setAwal(isian);
      setBerhasil(true);
    } catch (err) {
      setGalat(err?.response?.data?.error || 'Gagal menyimpan pengaturan surat.');
    } finally {
      setMenyimpan(false);
    }
  };

  if (memuat) return <LoadingSpinner />;

  const lengkap = form.kode_surat.trim() !== '' && form.penandatangan_nama.trim() !== '';

  return (
    <div className="space-y-4">
      {galat && <Alert variant="error" onClose={() => setGalat('')}>{galat}</Alert>}
      {berhasil && <Alert variant="success" onClose={() => setBerhasil(false)}>Pengaturan surat tersimpan.</Alert>}
      {!lengkap && <Alert variant="warning">Kode surat dan nama penandatangan wajib diisi agar jamaah bisa mengunduh surat.</Alert>}

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <Input label="Nama Perusahaan di Surat" name="nama_perusahaan" value={form.nama_perusahaan} onChange={ubah} placeholder="cth. PT. Hana Asia Wisata" className="!mb-0" />
        <Input label="Kode Surat" name="kode_surat" value={form.kode_surat} onChange={ubah} placeholder="cth. HAW" className="!mb-0" />
        <Input label="Nama Penandatangan" name="penandatangan_nama" value={form.penandatangan_nama} onChange={ubah} placeholder="cth. Fikry Shahab" className="!mb-0" />
        <Input label="Jabatan Penandatangan" name="penandatangan_jabatan" value={form.penandatangan_jabatan} onChange={ubah} placeholder="cth. Direktur Utama" className="!mb-0" />
        <Input
          label="Nomor Surat Terakhir"
          name="nomor_terakhir"
          type="number"
          min={awal.nomor_terakhir}
          value={form.nomor_terakhir}
          onChange={ubah}
          className="!mb-0"
        />
      </div>
      <p className="text-[12px] text-neutral-500 font-body">
        Nomor berikutnya = nomor terakhir + 1, mis. {form.nomor_terakhir + 1}/{form.kode_surat || 'KODE'}/SKU/XI/2026. Nomor tidak bisa diturunkan agar tidak ada nomor ganda.
        Kota surat diambil dari Kontak &amp; Alamat, dan nomor izin PPIU/PIHK dari Legalitas.
      </p>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {GAMBAR.map((g) => (
          <div key={g.kunci}>
            <FileInput
              label={g.label}
              value={form[g.kunci]}
              previewUrl={pratinjau[g.kunci]}
              onChange={unggah(g.kunci)}
              onRemove={hapus(g.kunci)}
              isUploading={mengunggah === g.kunci}
              uploadingText="Mengunggah..."
              placeholder="Pilih file..."
            />
            <p className="mt-1 text-[12px] text-neutral-500 font-body">{g.petunjuk}</p>
          </div>
        ))}
      </div>

      <div className="flex justify-end">
        <Button type="button" onClick={simpan} disabled={menyimpan || mengunggah !== ''}>
          {menyimpan ? 'Menyimpan...' : 'Simpan Pengaturan Surat'}
        </Button>
      </div>
    </div>
  );
};

export default BrandSurat;
