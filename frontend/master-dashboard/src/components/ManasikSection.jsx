import React, { useCallback, useEffect, useState } from 'react';
import { ExternalLink } from 'lucide-react';
import Modal from 'shared/ui/Modal';
import Input from 'shared/ui/Input';
import Textarea from 'shared/ui/Textarea';
import Button from 'shared/ui/Button';
import Alert from 'shared/ui/Alert';
import Badge from 'shared/ui/Badge';
import EmptyState from 'shared/ui/EmptyState';
import { Field } from 'shared/ui/DetailLayout';
import { getScheduleManasik, saveScheduleManasik, deleteScheduleManasik } from '../api/schedules';

const KOSONG = { judul: '', tanggal: '', jam_mulai: '', jam_selesai: '', agenda: '', ustadz: '', tempat: '', alamat: '', gmaps_url: '' };

const formatTanggal = (iso) =>
  new Date(`${iso}T00:00:00`).toLocaleDateString('id-ID', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' });

const jam = (hhmm) => hhmm.replace(':', '.');

// Manasik keberangkatan: satu per jadwal (keputusan 14 November 2026). Jamaah melihatnya di app Hana dan mengonfirmasi
// kehadiran per jamaah; rekap konfirmasi tampil di sini. Konfirmasi tidak mengubah penanda progres "sudah manasik".
const ManasikSection = ({ scheduleId, canEdit = true }) => {
  const [data, setData] = useState(null);
  const [loadError, setLoadError] = useState(null);
  const [form, setForm] = useState(null);
  const [formError, setFormError] = useState(null);
  const [isSaving, setIsSaving] = useState(false);
  const [konfirmasiHapus, setKonfirmasiHapus] = useState(false);
  const [hapusError, setHapusError] = useState(null);
  const [lihatJamaah, setLihatJamaah] = useState(false);

  const load = useCallback(async () => {
    try {
      setData(await getScheduleManasik(scheduleId));
      setLoadError(null);
    } catch (err) {
      setLoadError(err.response?.data?.error || 'Gagal memuat manasik.');
    }
  }, [scheduleId]);

  useEffect(() => { load(); }, [load]);

  if (loadError) return <Alert variant="error">{loadError}</Alert>;
  if (!data) return <p className="text-xs text-neutral-500 font-body">Memuat manasik...</p>;

  const m = data.manasik;
  const rekap = data.rekap;

  const bukaForm = () => {
    setForm(m ? {
      judul: m.judul, tanggal: m.tanggal, jam_mulai: m.jam_mulai, jam_selesai: m.jam_selesai,
      agenda: m.agenda.join('\n'), ustadz: m.ustadz || '', tempat: m.tempat, alamat: m.alamat, gmaps_url: m.gmaps_url || '',
    } : KOSONG);
    setFormError(null);
  };

  const ubah = (key) => (e) => setForm((f) => ({ ...f, [key]: e.target.value }));

  const simpan = async () => {
    setIsSaving(true);
    setFormError(null);
    try {
      setData(await saveScheduleManasik(scheduleId, { ...form, agenda: form.agenda.split('\n') }));
      setForm(null);
    } catch (err) {
      setFormError(err.response?.data?.error || 'Gagal menyimpan manasik.');
    } finally {
      setIsSaving(false);
    }
  };

  const hapus = async () => {
    setIsSaving(true);
    try {
      await deleteScheduleManasik(scheduleId);
      setKonfirmasiHapus(false);
      await load();
    } catch (err) {
      setHapusError(err.response?.data?.error || 'Gagal menghapus manasik.');
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="space-y-4 font-body">
      {m ? (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Field label="Judul">{m.judul}</Field>
            <Field label="Waktu">{formatTanggal(m.tanggal)}, {jam(m.jam_mulai)}–{jam(m.jam_selesai)} WIB</Field>
            <Field label="Ustadz pengisi">{m.ustadz || '-'}</Field>
            <Field label="Tempat">
              {m.tempat}
              <span className="block text-xs font-normal text-neutral-600">{m.alamat}</span>
              {m.gmaps_url && (
                <a href={m.gmaps_url} target="_blank" rel="noopener noreferrer" className="mt-1 inline-flex items-center gap-1 text-xs text-neutral-900 underline underline-offset-2 decoration-neutral-300 hover:decoration-neutral-900">
                  Buka Google Maps
                  <ExternalLink size={12} strokeWidth={1.75} />
                </a>
              )}
            </Field>
          </div>
          {m.agenda.length > 0 && (
            <div>
              <span className="block text-xs text-neutral-400 mb-1">Agenda</span>
              <ol className="list-decimal pl-5 space-y-0.5 text-sm text-neutral-900">
                {m.agenda.map((a, i) => <li key={i}>{a}</li>)}
              </ol>
            </div>
          )}
          {rekap && (
            <div className="pt-3 border-t border-neutral-200 space-y-2">
              <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-neutral-600">
                <span>Konfirmasi hadir:</span>
                <span><span className="font-semibold text-neutral-900 tabular-nums">{rekap.hadir}</span> hadir</span>
                <span><span className="font-semibold text-neutral-900 tabular-nums">{rekap.tidak_hadir}</span> tidak hadir</span>
                <span><span className="font-semibold text-neutral-900 tabular-nums">{rekap.belum}</span> belum menjawab</span>
              </div>
              {rekap.jamaah.length > 0 && (
                <details className="text-xs text-neutral-600" open={lihatJamaah} onToggle={(e) => setLihatJamaah(e.currentTarget.open)}>
                  <summary className="cursor-pointer font-medium text-neutral-700">Daftar jamaah ({rekap.jamaah.length})</summary>
                  <ul className="mt-2 divide-y divide-neutral-100">
                    {rekap.jamaah.map((j) => (
                      <li key={j.jamaah_id} className="py-1.5 flex items-center justify-between gap-3">
                        <span className="min-w-0 truncate text-neutral-900">
                          {j.nama} <span className="font-mono text-xs text-neutral-500">{j.booking_code}</span>
                        </span>
                        {j.hadir === true && <Badge variant="success">Hadir</Badge>}
                        {j.hadir === false && <Badge variant="danger">Tidak hadir</Badge>}
                        {j.hadir == null && <Badge variant="neutral">Belum menjawab</Badge>}
                      </li>
                    ))}
                  </ul>
                </details>
              )}
            </div>
          )}
          {canEdit && (
            <div className="flex gap-2">
              <Button variant="secondary" size="sm" onClick={bukaForm}>Ubah Manasik</Button>
              <Button variant="danger-outline" size="sm" onClick={() => { setHapusError(null); setKonfirmasiHapus(true); }}>Hapus</Button>
            </div>
          )}
        </>
      ) : (
        <>
          <EmptyState variant="inline" message="Manasik untuk keberangkatan ini belum diatur. Jamaah melihat keterangan &quot;belum diumumkan&quot; di app." />
          {canEdit && <Button variant="secondary" size="sm" onClick={bukaForm}>Atur Manasik</Button>}
        </>
      )}

      <Modal
        isOpen={form !== null}
        onClose={() => !isSaving && setForm(null)}
        title={m ? 'Ubah Manasik' : 'Atur Manasik'}
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setForm(null)} disabled={isSaving}>Batal</Button>
            <Button variant="primary" onClick={simpan} isLoading={isSaving}>Simpan</Button>
          </div>
        }
      >
        {form && (
          <div className="space-y-4">
            {formError && <Alert variant="error">{formError}</Alert>}
            <Input label="Judul" name="manasik_judul" required maxLength={150} value={form.judul} onChange={ubah('judul')} placeholder="mis. Manasik Umroh Lengkap" />
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <Input label="Tanggal" name="manasik_tanggal" type="date" required value={form.tanggal} onChange={ubah('tanggal')} />
              <Input label="Jam mulai" name="manasik_jam_mulai" type="time" required value={form.jam_mulai} onChange={ubah('jam_mulai')} />
              <Input label="Jam selesai" name="manasik_jam_selesai" type="time" required value={form.jam_selesai} onChange={ubah('jam_selesai')} />
            </div>
            <Textarea
              label="Agenda"
              name="manasik_agenda"
              rows={5}
              value={form.agenda}
              onChange={ubah('agenda')}
              helpText="Satu butir per baris, maksimal 30 butir."
              placeholder={'08.00 Pembukaan\n09.00 Praktik thawaf dan sa\'i'}
            />
            <Input label="Ustadz pengisi" name="manasik_ustadz" maxLength={150} value={form.ustadz} onChange={ubah('ustadz')} />
            <Input label="Nama tempat" name="manasik_tempat" required maxLength={150} value={form.tempat} onChange={ubah('tempat')} placeholder="mis. Aula Kantor Pusat" />
            <Textarea label="Alamat" name="manasik_alamat" required rows={2} maxLength={500} value={form.alamat} onChange={ubah('alamat')} />
            <Input
              label="Link Google Maps"
              name="manasik_gmaps"
              maxLength={500}
              value={form.gmaps_url}
              onChange={ubah('gmaps_url')}
              placeholder="https://maps.app.goo.gl/..."
              helpText="Opsional. Tanpa link, app membuka petunjuk arah ke alamat di atas."
            />
          </div>
        )}
      </Modal>

      <Modal
        isOpen={konfirmasiHapus}
        onClose={() => !isSaving && setKonfirmasiHapus(false)}
        title="Hapus Manasik"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setKonfirmasiHapus(false)} disabled={isSaving}>Batal</Button>
            <Button variant="danger" onClick={hapus} isLoading={isSaving}>Hapus</Button>
          </div>
        }
      >
        {hapusError && <div className="mb-3"><Alert variant="error">{hapusError}</Alert></div>}
        <p className="text-sm text-neutral-700 font-body">
          Manasik dan seluruh konfirmasi hadir jamaah untuk keberangkatan ini akan dihapus.
        </p>
      </Modal>
    </div>
  );
};

export default ManasikSection;
