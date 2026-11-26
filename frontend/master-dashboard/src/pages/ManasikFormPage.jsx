import React, { useContext, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import PageHeader from 'shared/ui/PageHeader';
import FormLayout, { FormSection, FormAside, SidebarSection } from 'shared/ui/FormLayout';
import Input from 'shared/ui/Input';
import Textarea from 'shared/ui/Textarea';
import Button from 'shared/ui/Button';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import { AuthContext } from '../context/AuthContext';
import PilihPaketManasik, { tanggalPendek } from '../components/PilihPaketManasik';
import { createManasik, getManasik, listPaketTersediaManasik, updateManasik } from '../api/manasik';
import { listBrandOptions } from '../api/brands';

const jamTitik = (hhmm) => (hhmm || '').replace(':', '.');

const KOSONG = { judul: '', tanggal: '', jam_mulai: '', jam_selesai: '', agenda: '', ustadz: '', tempat: '', alamat: '', gmaps_url: '' };

// Tambah/ubah manasik. Saat tambah, admin sekaligus memilih paket yang ikut; paket berikutnya ditambah atau dilepas di
// halaman detail. Manasik yang disimpan langsung tampil di app Hana untuk jamaah paket-paketnya.
const ManasikFormPage = () => {
  const { id } = useParams();
  const isEdit = Boolean(id);
  const navigate = useNavigate();
  const { isHolding } = useContext(AuthContext);
  const [form, setForm] = useState(KOSONG);
  const [paket, setPaket] = useState([]);
  const [scheduleIds, setScheduleIds] = useState([]);
  const [brandNama, setBrandNama] = useState({});
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState(null);
  // Mulai 00.00 WIB hari manasik, tanggal dan jam terkunci (tampil sebagai teks).
  const [terkunci, setTerkunci] = useState(false);
  const hariIni = new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Jakarta' });

  useEffect(() => {
    let batal = false;
    const muat = async () => {
      try {
        if (isEdit) {
          const d = await getManasik(id);
          const m = d.manasik;
          if (!batal) {
            setTerkunci(Boolean(d.sudah_dimulai));
            setForm({
              judul: m.judul, tanggal: m.tanggal, jam_mulai: m.jam_mulai, jam_selesai: m.jam_selesai,
              agenda: m.agenda.join('\n'), ustadz: m.ustadz || '', tempat: m.tempat, alamat: m.alamat, gmaps_url: m.gmaps_url || '',
            });
          }
        } else {
          const [daftar, brands] = await Promise.all([
            listPaketTersediaManasik(),
            isHolding ? listBrandOptions().catch(() => []) : Promise.resolve([]),
          ]);
          if (!batal) {
            setPaket(daftar || []);
            setBrandNama(Object.fromEntries((brands || []).map((b) => [b.id, b.name])));
          }
        }
      } catch (err) {
        if (!batal) setErrorMessage(err.response?.data?.error || 'Gagal memuat data manasik.');
      } finally {
        if (!batal) setIsLoading(false);
      }
    };
    muat();
    return () => { batal = true; };
  }, [id, isEdit, isHolding]);

  const ubah = (key) => (e) => setForm((f) => ({ ...f, [key]: e.target.value }));

  const simpan = async () => {
    setIsSaving(true);
    setErrorMessage(null);
    const payload = { ...form, agenda: form.agenda.split('\n') };
    try {
      if (isEdit) {
        await updateManasik(id, payload);
        navigate(`/manasik/${id}`, { state: { successMessage: 'Manasik disimpan.' } });
      } else {
        const d = await createManasik({ ...payload, schedule_ids: scheduleIds });
        navigate(`/manasik/${d.manasik.id}`, { state: { successMessage: 'Manasik dibuat.' } });
      }
    } catch (err) {
      setErrorMessage(err.response?.data?.error || 'Gagal menyimpan manasik.');
      setIsSaving(false);
    }
  };

  const kembali = () => navigate(isEdit ? `/manasik/${id}` : '/manasik');

  if (isLoading) {
    return (
      <div className="space-y-6">
        <PageHeader title={isEdit ? 'Ubah Manasik' : 'Buat Manasik'} onBack={kembali} />
        <LoadingSpinner variant="card" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader title={isEdit ? 'Ubah Manasik' : 'Buat Manasik'} onBack={kembali} />
      {errorMessage && <Alert variant="error" onClose={() => setErrorMessage(null)}>{errorMessage}</Alert>}
      <FormLayout>
        <FormSection title="Informasi Manasik">
          <div className="space-y-4">
            <Input label="Judul" name="judul" required maxLength={150} value={form.judul} onChange={ubah('judul')} placeholder="Manasik Umroh Lengkap" />
            {terkunci ? (
              <div>
                <div className="text-sm font-medium font-body text-neutral-800">Tanggal dan jam</div>
                <div className="mt-0.5 text-sm font-body text-neutral-900">
                  {tanggalPendek(form.tanggal)}, {jamTitik(form.jam_mulai)}–{jamTitik(form.jam_selesai)} WIB
                </div>
              </div>
            ) : (
              <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                <Input
                  label="Tanggal"
                  name="tanggal"
                  type="date"
                  required
                  min={hariIni}
                  value={form.tanggal}
                  onChange={ubah('tanggal')}
                />
                <Input label="Jam mulai" name="jam_mulai" type="time" required value={form.jam_mulai} onChange={ubah('jam_mulai')} />
                <Input label="Jam selesai" name="jam_selesai" type="time" required value={form.jam_selesai} onChange={ubah('jam_selesai')} />
              </div>
            )}
            <Input label="Ustadz pengisi" name="ustadz" maxLength={150} value={form.ustadz} onChange={ubah('ustadz')} />
            <Textarea
              label="Agenda"
              name="agenda"
              rows={6}
              value={form.agenda}
              onChange={ubah('agenda')}
              placeholder={'08.00 Pembukaan\n09.00 Praktik thawaf dan sa\'i'}
            />
          </div>
        </FormSection>
        <FormSection title="Lokasi">
          <div className="space-y-4">
            <Input label="Nama tempat" name="tempat" required maxLength={150} value={form.tempat} onChange={ubah('tempat')} placeholder="Aula Kantor Pusat" />
            <Textarea label="Alamat" name="alamat" required rows={2} maxLength={500} value={form.alamat} onChange={ubah('alamat')} />
            <Input
              label="Link Google Maps"
              name="gmaps_url"
              maxLength={500}
              value={form.gmaps_url}
              onChange={ubah('gmaps_url')}
              placeholder="https://maps.app.goo.gl/..."
            />
          </div>
        </FormSection>
        {!isEdit && (
          <FormSection title="Paket">
            <PilihPaketManasik
              paket={paket}
              value={scheduleIds}
              onChange={setScheduleIds}
              tanggalManasik={form.tanggal}
              brandNama={brandNama}
              tampilBrand={isHolding}
            />
          </FormSection>
        )}
        <FormAside>
          <SidebarSection>
            <Button className="w-full" onClick={simpan} isLoading={isSaving} disabled={!isEdit && scheduleIds.length === 0}>
              Simpan
            </Button>
          </SidebarSection>
        </FormAside>
      </FormLayout>
    </div>
  );
};

export default ManasikFormPage;
