import React, { useContext, useEffect, useState } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import PageHeader from 'shared/ui/PageHeader';
import FormLayout, { FormSection, FormAside, SidebarSection } from 'shared/ui/FormLayout';
import Input from 'shared/ui/Input';
import FileInput from 'shared/ui/FileInput';
import Toggle from 'shared/ui/Toggle';
import Button from 'shared/ui/Button';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import { AuthContext } from '../context/AuthContext';
import TujuanPromosiField from '../components/TujuanPromosiField';
import { mediaSrc } from './BannerPromosiPage';
import { createBannerPromosi, listBannerPromosi, updateBannerPromosi } from '../api/bannerPromosi';
import { listSchedulesAdmin } from '../api/schedules';
import { uploadMedia } from '../api/media';
import PotongGambarModal from '../components/PotongGambarModal';
import { bacaGambar, cukupBesar, lepasGambar, pas21, siapkanGambar21 } from '../utils/gambarBanner';

const KOSONG = { judul: '', gambar_url: '', tujuan: 'tidak_ada', tujuan_nilai: '', tampil_di: 'keduanya', mulai: '', selesai: '', aktif: true };
// Dua kotak centang (Website, Aplikasi) ke satu nilai: web | app | keduanya. Minimal satu tetap tercentang.
const ubahTampil = (sekarang, kode, centang) => {
  const lain = kode === 'web' ? 'app' : 'web';
  const adaLain = sekarang === 'keduanya' || sekarang === lain;
  if (centang) return adaLain ? 'keduanya' : kode;
  return adaLain ? lain : sekarang;
};

const BannerPromosiFormPage = () => {
  const { id } = useParams();
  const isEdit = Boolean(id);
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { isHolding } = useContext(AuthContext);
  const brandId = isHolding ? searchParams.get('brand') || '' : '';
  const [form, setForm] = useState(KOSONG);
  const [paket, setPaket] = useState([]);
  const [pratinjau, setPratinjau] = useState(null);
  const [mengunggah, setMengunggah] = useState(false);
  const [potong, setPotong] = useState(null); // { img, file } saat gambar perlu dipotong ke 2:1
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState(null);
  const kembali = () => navigate(`/banner-promosi${brandId ? `?brand=${brandId}` : ''}`);

  useEffect(() => {
    let batal = false;
    (async () => {
      try {
        const [d, s] = await Promise.all([isEdit ? listBannerPromosi(brandId) : Promise.resolve(null), listSchedulesAdmin().catch(() => [])]);
        if (batal) return;
        const brandAktif = brandId ? Number(brandId) : null;
        setPaket((s || []).filter((p) => p.status !== 'archived' && (!brandAktif || p.brand_id === brandAktif)));
        if (isEdit) {
          const b = d.banner.find((x) => String(x.id) === String(id));
          if (!b) throw new Error('Banner tidak ditemukan.');
          setForm({ judul: b.judul, gambar_url: b.gambar_url, tujuan: b.tujuan, tujuan_nilai: b.tujuan_nilai || '', tampil_di: b.tampil_di || 'keduanya', mulai: b.mulai || '', selesai: b.selesai || '', aktif: b.aktif });
          setPratinjau(mediaSrc(b.gambar_url));
        }
      } catch (err) {
        if (!batal) setErrorMessage(err.response?.data?.error || err.message || 'Gagal memuat banner.');
      } finally {
        if (!batal) setIsLoading(false);
      }
    })();
    return () => { batal = true; };
  }, [id, isEdit, brandId]);

  const unggah = async (berkas) => {
    setPratinjau(URL.createObjectURL(berkas));
    setMengunggah(true);
    try {
      const url = await uploadMedia(berkas, 'banner-promosi');
      setForm((f) => ({ ...f, gambar_url: url }));
    } catch (err) {
      setErrorMessage(err.response?.data?.error || 'Gagal mengunggah gambar.');
      setPratinjau(null);
    } finally {
      setMengunggah(false);
    }
  };

  // Gambar 2:1 langsung diunggah (dikecilkan dan dikompres bila perlu); rasio lain dipotong dulu.
  const pilihGambar = async (e) => {
    const file = e.target.files?.[0];
    e.target.value = '';
    if (!file) return;
    setErrorMessage(null);
    let img;
    try {
      img = await bacaGambar(file);
      if (!cukupBesar(img)) throw new Error('Gambar minimal 1200 × 600 px');
      if (pas21(img)) {
        const siap = await siapkanGambar21(file, img);
        lepasGambar(img);
        await unggah(siap);
      } else setPotong({ img, file });
    } catch (err) {
      lepasGambar(img);
      setErrorMessage(err.message);
    }
  };

  const simpan = async () => {
    setIsSaving(true);
    setErrorMessage(null);
    const payload = { ...form, mulai: form.mulai || null, selesai: form.selesai || null, brand_id: brandId ? Number(brandId) : undefined };
    try {
      if (isEdit) await updateBannerPromosi(id, payload);
      else await createBannerPromosi(payload);
      navigate(`/banner-promosi${brandId ? `?brand=${brandId}` : ''}`, { state: { successMessage: isEdit ? 'Banner disimpan.' : 'Banner ditambahkan.' } });
    } catch (err) {
      setErrorMessage(err.response?.data?.error || 'Gagal menyimpan banner.');
      setIsSaving(false);
    }
  };

  const ubah = (key) => (e) => setForm((f) => ({ ...f, [key]: e.target.value }));
  const judul = isEdit ? 'Ubah Banner' : 'Tambah Banner';

  if (isLoading) {
    return (
      <div className="space-y-6">
        <PageHeader title={judul} onBack={kembali} />
        <LoadingSpinner variant="card" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader title={judul} onBack={kembali} />
      {errorMessage && <Alert variant="error" onClose={() => setErrorMessage(null)}>{errorMessage}</Alert>}
      <FormLayout>
        <FormSection title="Banner">
          <div className="space-y-4">
            <Input label="Judul" name="judul" required maxLength={100} value={form.judul} onChange={ubah('judul')} placeholder="Early Bird Desember" />
            <FileInput
              label="Gambar"
              accept="image/jpeg,image/png,image/webp"
              value={form.gambar_url}
              previewUrl={pratinjau}
              onChange={pilihGambar}
              onRemove={() => { setPratinjau(null); setForm((f) => ({ ...f, gambar_url: '' })); }}
              isUploading={mengunggah}
              uploadingText="Mengunggah gambar..."
              helperText="1200 × 600 px (rasio 2:1), JPG/PNG/WebP"
              placeholder="Pilih gambar..."
            />
          </div>
        </FormSection>
        <FormSection title="Tampil di">
          <div className="flex flex-wrap gap-6">
            {[['web', 'Website'], ['app', 'Aplikasi']].map(([kode, label]) => (
              <label key={kode} className="flex items-center gap-2 font-body text-sm text-neutral-800">
                <input
                  type="checkbox"
                  className="h-4 w-4 shrink-0 rounded border-neutral-300 accent-primary-500"
                  checked={form.tampil_di === 'keduanya' || form.tampil_di === kode}
                  onChange={(e) => setForm((f) => ({ ...f, tampil_di: ubahTampil(f.tampil_di, kode, e.target.checked) }))}
                />
                {label}
              </label>
            ))}
          </div>
        </FormSection>
        <FormSection title="Tujuan">
          <TujuanPromosiField tujuan={form.tujuan} nilai={form.tujuan_nilai} paket={paket} onChange={(t) => setForm((f) => ({ ...f, ...t }))} />
        </FormSection>
        <FormSection title="Jadwal Tayang">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <Input label="Mulai" name="mulai" type="date" value={form.mulai} onChange={ubah('mulai')} className="!mb-0" />
            <Input label="Selesai" name="selesai" type="date" value={form.selesai} onChange={ubah('selesai')} className="!mb-0" />
          </div>
        </FormSection>
        <FormAside>
          <SidebarSection>
            <Button className="w-full" onClick={simpan} isLoading={isSaving} disabled={mengunggah || !form.gambar_url || !form.judul.trim()}>
              Simpan Banner
            </Button>
          </SidebarSection>
          <SidebarSection title="Status">
            <label htmlFor="banner_aktif" className="flex items-center justify-between gap-3 font-body text-sm text-neutral-800">
              Aktif
              <Toggle id="banner_aktif" name="aktif" checked={form.aktif} onChange={(e) => setForm((f) => ({ ...f, aktif: e.target.checked }))} />
            </label>
          </SidebarSection>
        </FormAside>
      </FormLayout>
      {potong && (
        <PotongGambarModal
          img={potong.img}
          file={potong.file}
          onBatal={() => {
            lepasGambar(potong.img);
            setPotong(null);
          }}
          onSelesai={(hasil) => {
            lepasGambar(potong.img);
            setPotong(null);
            unggah(hasil);
          }}
        />
      )}
    </div>
  );
};

export default BannerPromosiFormPage;
