import React, { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Plus, X } from 'lucide-react';
import PageHeader from 'shared/ui/PageHeader';
import FormLayout, { FormSection, FormAside, SidebarSection } from 'shared/ui/FormLayout';
import Input from 'shared/ui/Input';
import Textarea from 'shared/ui/Textarea';
import CustomDropdown from 'shared/ui/CustomDropdown';
import FileInput from 'shared/ui/FileInput';
import Toggle from 'shared/ui/Toggle';
import Button from 'shared/ui/Button';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import { KATEGORI_HARAMAIN, KOTA_HARAMAIN, mediaSrc } from './HaramainPage';
import { createHaramain, getHaramain, updateHaramain } from '../api/haramain';
import { uploadMedia } from '../api/media';
import { kompresFoto } from '../utils/kompresFoto';
import EditorTeks from '../components/EditorTeks';

const KOSONG = {
  nama: '', nama_lengkap: '', kategori: 'masjid', kota: 'makkah', foto_url: '', galeri: [], ringkas: '', deskripsi: '',
  alamat: '', jam_buka: '', gmaps_url: '', populer: false, aktif: true,
};
const MAKS_GALERI = 10;
const valueOf = (v) => (v?.target ? v.target.value : v);

// Tambah/ubah tempat Jelajah Haramain. Foto dikecilkan (lebar maks 1600 px) dan dikompres sebelum diunggah.
const HaramainFormPage = () => {
  const { id } = useParams();
  const isEdit = Boolean(id);
  const navigate = useNavigate();
  const [form, setForm] = useState(KOSONG);
  const [isLoading, setIsLoading] = useState(isEdit);
  const [isSaving, setIsSaving] = useState(false);
  const [mengunggah, setMengunggah] = useState(null); // 'utama' | 'galeri'
  const [errorMessage, setErrorMessage] = useState(null);
  const kembali = () => navigate('/haramain');

  useEffect(() => {
    if (!isEdit) return;
    getHaramain(id)
      .then((t) => setForm({
        nama: t.nama, nama_lengkap: t.nama_lengkap || '', kategori: t.kategori, kota: t.kota, foto_url: t.foto_url, galeri: t.galeri || [],
        ringkas: t.ringkas, deskripsi: t.deskripsi || '', alamat: t.alamat || '', jam_buka: t.jam_buka || '', gmaps_url: t.gmaps_url || '',
        populer: t.populer, aktif: t.aktif,
      }))
      .catch((err) => setErrorMessage(err.response?.data?.error || 'Gagal memuat tempat.'))
      .finally(() => setIsLoading(false));
  }, [id, isEdit]);

  const ubah = (key) => (e) => setForm((f) => ({ ...f, [key]: valueOf(e) }));

  const unggah = async (file) => uploadMedia(await kompresFoto(file), 'haramain');

  const pilihUtama = async (e) => {
    const file = e.target.files?.[0];
    e.target.value = '';
    if (!file) return;
    setErrorMessage(null);
    setMengunggah('utama');
    try {
      const url = await unggah(file);
      setForm((f) => ({ ...f, foto_url: url }));
    } catch (err) {
      setErrorMessage(err.response?.data?.error || err.message || 'Gagal mengunggah foto.');
    } finally {
      setMengunggah(null);
    }
  };

  const pilihGaleri = async (e) => {
    const files = [...(e.target.files || [])].slice(0, MAKS_GALERI - form.galeri.length);
    e.target.value = '';
    if (files.length === 0) return;
    setErrorMessage(null);
    setMengunggah('galeri');
    try {
      const urls = [];
      for (const f of files) urls.push(await unggah(f));
      setForm((x) => ({ ...x, galeri: [...x.galeri, ...urls] }));
    } catch (err) {
      setErrorMessage(err.response?.data?.error || err.message || 'Gagal mengunggah foto.');
    } finally {
      setMengunggah(null);
    }
  };

  const simpan = async () => {
    setIsSaving(true);
    setErrorMessage(null);
    try {
      if (isEdit) await updateHaramain(id, form);
      else await createHaramain(form);
      navigate('/haramain', { state: { successMessage: isEdit ? 'Tempat disimpan.' : 'Tempat ditambahkan.' } });
    } catch (err) {
      setErrorMessage(err.response?.data?.error || 'Gagal menyimpan tempat.');
      setIsSaving(false);
    }
  };

  const judul = isEdit ? 'Ubah Tempat' : 'Tambah Tempat';
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
        <FormSection title="Informasi">
          <div className="space-y-4">
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <Input label="Nama" name="nama" required maxLength={100} value={form.nama} onChange={ubah('nama')} placeholder="Masjid an-Nabawi" className="!mb-0" />
              <Input label="Nama lengkap" name="nama_lengkap" maxLength={150} value={form.nama_lengkap} onChange={ubah('nama_lengkap')} className="!mb-0" />
              <CustomDropdown label="Kategori" value={form.kategori} onChange={ubah('kategori')} options={KATEGORI_HARAMAIN} className="!mb-0" />
              <CustomDropdown label="Kota" value={form.kota} onChange={ubah('kota')} options={KOTA_HARAMAIN} className="!mb-0" />
            </div>
            <Input label="Ringkasan" name="ringkas" required maxLength={100} value={form.ringkas} onChange={ubah('ringkas')} placeholder="Masjid Nabi" />
            <EditorTeks
              label="Deskripsi"
              value={form.deskripsi}
              onChange={(html) => setForm((f) => ({ ...f, deskripsi: html }))}
              placeholder="Tulis tentang lokasi ini..."
              helpText="Judul untuk tiap bagian, daftar untuk poin. Tampil rapi di app."
              minTinggi="min-h-64"
            />
          </div>
        </FormSection>
        <FormSection title="Foto">
          <div className="space-y-4">
            <FileInput
              label="Foto utama"
              accept="image/jpeg,image/png,image/webp"
              value={form.foto_url}
              previewUrl={mediaSrc(form.foto_url)}
              onChange={pilihUtama}
              onRemove={() => setForm((f) => ({ ...f, foto_url: '' }))}
              isUploading={mengunggah === 'utama'}
              uploadingText="Mengunggah foto..."
              placeholder="Pilih foto..."
            />
            <div>
              <div className="mb-1.5 text-sm font-medium text-neutral-800 font-body">Galeri ({form.galeri.length}/{MAKS_GALERI})</div>
              <div className="flex flex-wrap gap-3">
                {form.galeri.map((g) => (
                  <div key={g} className="relative">
                    <img src={mediaSrc(g)} alt="" className="h-20 w-28 rounded-md border border-neutral-200 object-cover" />
                    <button
                      type="button"
                      aria-label="Hapus foto"
                      onClick={() => setForm((f) => ({ ...f, galeri: f.galeri.filter((x) => x !== g) }))}
                      className="absolute -right-2 -top-2 flex h-6 w-6 items-center justify-center rounded-full border border-neutral-200 bg-white text-neutral-600 hover:text-danger-600"
                    >
                      <X size={14} />
                    </button>
                  </div>
                ))}
                {form.galeri.length < MAKS_GALERI && (
                  <label className="flex h-20 w-28 cursor-pointer items-center justify-center rounded-md border border-dashed border-neutral-300 text-neutral-500 hover:border-neutral-400 hover:text-neutral-700">
                    {mengunggah === 'galeri' ? <LoadingSpinner variant="inline" /> : <Plus size={16} />}
                    <input type="file" accept="image/jpeg,image/png,image/webp" multiple className="hidden" onChange={pilihGaleri} disabled={mengunggah !== null} />
                  </label>
                )}
              </div>
            </div>
          </div>
        </FormSection>
        <FormSection title="Lokasi">
          <div className="space-y-4">
            <Textarea label="Alamat" name="alamat" rows={2} maxLength={300} value={form.alamat} onChange={ubah('alamat')} />
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <Input label="Jam buka" name="jam_buka" maxLength={60} value={form.jam_buka} onChange={ubah('jam_buka')} placeholder="Buka 24 jam" className="!mb-0" />
              <Input label="Link Google Maps" name="gmaps_url" maxLength={500} value={form.gmaps_url} onChange={ubah('gmaps_url')} placeholder="https://maps.app.goo.gl/..." className="!mb-0" />
            </div>
          </div>
        </FormSection>
        <FormAside>
          <SidebarSection>
            <Button className="w-full" onClick={simpan} isLoading={isSaving} disabled={mengunggah !== null || !form.foto_url || !form.nama.trim() || !form.ringkas.trim()}>
              Simpan Tempat
            </Button>
          </SidebarSection>
          <SidebarSection title="Status">
            <div className="space-y-3 font-body text-sm text-neutral-800">
              <label htmlFor="haramain_aktif" className="flex items-center justify-between gap-3">
                Aktif
                <Toggle id="haramain_aktif" name="aktif" checked={form.aktif} onChange={(e) => setForm((f) => ({ ...f, aktif: e.target.checked }))} />
              </label>
              <label htmlFor="haramain_populer" className="flex items-center justify-between gap-3">
                Populer
                <Toggle id="haramain_populer" name="populer" checked={form.populer} onChange={(e) => setForm((f) => ({ ...f, populer: e.target.checked }))} />
              </label>
            </div>
          </SidebarSection>
        </FormAside>
      </FormLayout>
    </div>
  );
};

export default HaramainFormPage;
