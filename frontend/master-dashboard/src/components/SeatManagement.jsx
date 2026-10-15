import React, { useState, useEffect, useCallback } from 'react';
import Modal from 'shared/ui/Modal';
import Input from 'shared/ui/Input';
import Textarea from 'shared/ui/Textarea';
import Button from 'shared/ui/Button';
import Alert from 'shared/ui/Alert';
import { getScheduleSeat, updateScheduleSeat } from '../api/schedules';

const formatWaktu = (iso) => new Date(iso).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short', timeZone: 'Asia/Jakarta' });

// Kuota & kursi sebuah paket: ringkasan, selisih terhadap booking, penyesuaian manual
// dengan alasan wajib, dan riwayatnya (SM-02, SM-03).
const SeatManagement = ({ scheduleId, bookingCount, canEdit = true }) => {
  const [overview, setOverview] = useState(null);
  const [loadError, setLoadError] = useState(null);
  const [isOpen, setIsOpen] = useState(false);
  const [nilai, setNilai] = useState('');
  const [alasan, setAlasan] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [formError, setFormError] = useState(null);

  const load = useCallback(async () => {
    try {
      setOverview(await getScheduleSeat(scheduleId));
      setLoadError(null);
    } catch (err) {
      setLoadError(err.response?.data?.error || 'Gagal memuat data kursi.');
    }
  }, [scheduleId]);

  useEffect(() => { load(); }, [load]);

  if (loadError) return <Alert variant="error">{loadError}</Alert>;
  if (!overview) return <p className="text-xs text-neutral-500 font-body">Memuat data kursi...</p>;

  const terisi = Math.max(0, overview.seat_total - overview.seat_sisa);
  const persen = overview.seat_total > 0 ? (terisi / overview.seat_total) * 100 : 0;
  const diLuar = overview.seat_di_luar_booking;

  const bukaForm = () => {
    setNilai(String(overview.seat_sisa));
    setAlasan('');
    setFormError(null);
    setIsOpen(true);
  };

  const simpan = async () => {
    const sisaBaru = Number(nilai);
    if (nilai === '' || !Number.isInteger(sisaBaru) || sisaBaru < 0 || sisaBaru > overview.seat_sisa_maks) {
      setFormError(`Sisa seat harus bilangan bulat antara 0 dan ${overview.seat_sisa_maks}.`);
      return;
    }
    if (sisaBaru === overview.seat_sisa) {
      setFormError('Nilainya sama dengan sisa seat saat ini.');
      return;
    }
    if (!alasan.trim()) {
      setFormError('Alasan penyesuaian wajib diisi.');
      return;
    }
    setIsSaving(true);
    setFormError(null);
    try {
      await updateScheduleSeat(scheduleId, sisaBaru, overview.seat_sisa, alasan.trim());
      setIsOpen(false);
      await load();
    } catch (err) {
      setFormError(err.response?.data?.error || 'Gagal menyimpan penyesuaian.');
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="space-y-4 font-body">
      <div className="grid grid-cols-2 gap-3 divide-x divide-neutral-200">
        <div>
          <span className="text-xs text-neutral-500 block">Sisa Seat</span>
          <span className="text-xl font-bold font-heading text-neutral-900">{overview.seat_sisa}</span>
        </div>
        <div className="pl-4">
          <span className="text-xs text-neutral-500 block">Total Kuota</span>
          <span className="text-xl font-bold font-heading text-neutral-900">{overview.seat_total}</span>
        </div>
      </div>

      {overview.seat_total > 0 && (
        <div className="space-y-1.5">
          <div className="flex justify-between text-xs text-neutral-600">
            <span>Terisi: {terisi} pax</span>
            <span>{Math.round(persen)}%</span>
          </div>
          <div className="w-full h-2.5 bg-neutral-100 rounded-full overflow-hidden border border-neutral-200">
            <div className="h-full bg-primary-600 rounded-full transition-all duration-300" style={{ width: `${Math.min(100, persen)}%` }} />
          </div>
        </div>
      )}

      <div className="pt-2 border-t border-neutral-200 text-xs text-neutral-600 space-y-1.5">
        <div className="flex justify-between">
          <span>Total Booking Terdaftar:</span>
          <span className="font-semibold text-neutral-900">{bookingCount || 0} booking</span>
        </div>
        <div className="flex justify-between">
          <span>Kursi dipegang booking:</span>
          <span className="font-semibold text-neutral-900">{overview.seat_dialokasikan} pax</span>
        </div>
        <div className="flex justify-between">
          <span>Terpakai di luar booking:</span>
          <span className="font-semibold text-neutral-900">{diLuar} pax</span>
        </div>
      </div>

      {diLuar > 0 && (
        <p className="text-xs text-neutral-500">
          {diLuar} kursi terpakai tanpa booking di ERP (impor data atau penjualan di luar sistem). Jika angka ini tidak benar, sesuaikan sisa seat.
        </p>
      )}
      {diLuar < 0 && (
        <Alert variant="warning">
          Sisa seat lebih besar {Math.abs(diLuar)} dari seharusnya: ada kursi dipegang booking yang belum dikurangkan. Sesuaikan sisa seat agar tidak terjadi penjualan melebihi kapasitas.
        </Alert>
      )}

      {canEdit && <Button variant="secondary" size="sm" onClick={bukaForm}>Sesuaikan Sisa Seat</Button>}

      {overview.adjustments.length > 0 && (
        <details className="text-xs text-neutral-600">
          <summary className="cursor-pointer font-medium text-neutral-700">Riwayat penyesuaian ({overview.adjustments.length})</summary>
          <ul className="mt-2 space-y-2">
            {overview.adjustments.map(a => (
              <li key={a.id} className="p-2 bg-neutral-50 border border-neutral-200 rounded-md">
                <div className="flex justify-between gap-2">
                  <span className="font-semibold text-neutral-900">{a.seat_before} → {a.seat_after}</span>
                  <span className="text-neutral-500">{formatWaktu(a.created_at)}</span>
                </div>
                <div className="mt-0.5">{a.reason}</div>
                <div className="mt-0.5 text-neutral-500">oleh {a.admin_email || 'akun yang sudah dihapus'}</div>
              </li>
            ))}
          </ul>
        </details>
      )}

      <Modal
        isOpen={isOpen}
        onClose={() => !isSaving && setIsOpen(false)}
        title="Sesuaikan Sisa Seat"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="ghost" onClick={() => setIsOpen(false)} disabled={isSaving}>Batal</Button>
            <Button variant="primary" onClick={simpan} isLoading={isSaving}>Simpan</Button>
          </div>
        }
      >
        <div className="space-y-4">
          {formError && <Alert variant="error">{formError}</Alert>}
          <Input
            label="Sisa seat baru"
            name="seat_sisa_baru"
            type="number"
            min="0"
            max={overview.seat_sisa_maks}
            value={nilai}
            onChange={(e) => setNilai(e.target.value)}
          />
          {overview.seat_sisa !== overview.seat_sisa_maks && (
            <button type="button" className="text-xs text-primary-700 underline" onClick={() => setNilai(String(overview.seat_sisa_maks))}>
              Isi sesuai booking ({overview.seat_sisa_maks})
            </button>
          )}
          <Textarea
            label="Alasan"
            name="alasan_seat"
            required
            maxLength={500}
            value={alasan}
            onChange={(e) => setAlasan(e.target.value)}
            placeholder="mis. 2 kursi terjual langsung tanpa booking"
          />
        </div>
      </Modal>
    </div>
  );
};

export default SeatManagement;
