import React, { useState, useEffect, useCallback } from 'react';
import Modal from 'shared/ui/Modal';
import Input from 'shared/ui/Input';
import Button from 'shared/ui/Button';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import { getScheduleSeat, samakanScheduleSeat, updateScheduleSeatTampil } from '../api/schedules';

const teksTampil = (n) => (n ? `${n} seat` : 'Mati');

const formatWaktu = (iso) => new Date(iso).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short', timeZone: 'Asia/Jakarta' });

const linkClass = 'font-medium text-neutral-900 underline underline-offset-2 decoration-neutral-300 hover:decoration-neutral-900';

// Kuota & seat sebuah paket. Tidak ada booking di luar sistem: sisa seat selalu kuota dikurangi
// seat yang dipegang booking. Selisih (bug atau data lama) hanya bisa disamakan dengan booking.
const SeatManagement = ({ scheduleId, canEdit = true }) => {
  const [overview, setOverview] = useState(null);
  const [loadError, setLoadError] = useState(null);
  const [samakanError, setSamakanError] = useState(null);
  const [isSamakan, setIsSamakan] = useState(false);
  const [isTampilOpen, setIsTampilOpen] = useState(false);
  const [nilaiTampil, setNilaiTampil] = useState('');
  const [tampilError, setTampilError] = useState(null);
  const [isSavingTampil, setIsSavingTampil] = useState(false);

  const load = useCallback(async () => {
    try {
      setOverview(await getScheduleSeat(scheduleId));
      setLoadError(null);
    } catch (err) {
      setLoadError(err.response?.data?.error || 'Data seat gagal dimuat. Coba lagi.');
    }
  }, [scheduleId]);

  useEffect(() => { load(); }, [load]);

  if (loadError) return <Alert variant="error">{loadError}</Alert>;
  if (!overview) return <LoadingSpinner />;

  // Terisi = seat dipegang booking (hitungan server), tetap benar saat sisa seat tidak cocok.
  const terisi = overview.seat_dialokasikan ?? Math.max(0, overview.seat_total - overview.seat_sisa);
  const persen = overview.seat_total > 0 ? (terisi / overview.seat_total) * 100 : 0;

  const samakan = async () => {
    setIsSamakan(true);
    setSamakanError(null);
    try {
      await samakanScheduleSeat(scheduleId);
      await load();
    } catch (err) {
      setSamakanError(err.response?.data?.error || 'Sisa seat gagal disamakan. Coba lagi.');
    } finally {
      setIsSamakan(false);
    }
  };

  // Seat tampil di publik (scarcity): hanya label, tidak mengubah sisa seat.
  const angkaTampil = overview.seat_tampil == null ? null : Math.min(overview.seat_tampil, overview.seat_sisa);

  const bukaTampil = () => {
    setNilaiTampil(overview.seat_tampil == null ? '' : String(overview.seat_tampil));
    setTampilError(null);
    setIsTampilOpen(true);
  };

  const simpanTampil = async (matikan = false) => {
    let angka = null;
    if (!matikan && nilaiTampil !== '') {
      angka = Number(nilaiTampil);
      if (!Number.isInteger(angka) || angka < 1 || angka > overview.seat_total) {
        setTampilError(`Seat tampil harus antara 1 dan ${overview.seat_total}.`);
        return;
      }
    }
    setIsSavingTampil(true);
    setTampilError(null);
    try {
      await updateScheduleSeatTampil(scheduleId, angka);
      setIsTampilOpen(false);
      await load();
    } catch (err) {
      setTampilError(err.response?.data?.error || 'Seat tampil gagal disimpan. Coba lagi.');
    } finally {
      setIsSavingTampil(false);
    }
  };

  return (
    <div className="space-y-4 font-body">
      <div className="grid grid-cols-2 gap-3 divide-x divide-neutral-200">
        <div>
          <span className="text-xs text-neutral-500 block">Sisa seat</span>
          <span className="text-2xl md:text-3xl font-semibold font-heading tracking-tight tabular-nums text-neutral-900">{overview.seat_sisa}</span>
        </div>
        <div className="pl-4">
          <span className="text-xs text-neutral-500 block">Kuota seat</span>
          <span className="text-2xl md:text-3xl font-semibold font-heading tracking-tight tabular-nums text-neutral-900">{overview.seat_total}</span>
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

      {!overview.seat_cocok && (
        <Alert variant="warning">
          <span className="flex flex-wrap items-center justify-between gap-2">
            <span>Sisa seat tidak cocok dengan booking.</span>
            {canEdit && (
              <button type="button" className={linkClass} onClick={samakan} disabled={isSamakan}>
                Samakan dengan booking
              </button>
            )}
          </span>
        </Alert>
      )}
      {samakanError && <Alert variant="error">{samakanError}</Alert>}

      <div className="pt-2 border-t border-neutral-200 text-xs text-neutral-600 space-y-1.5">
        <div className="flex justify-between gap-2">
          <span>Seat tampil:</span>
          <span className="flex items-center gap-2">
            <span className="font-semibold text-neutral-900">{angkaTampil == null ? 'Sesuai sisa seat' : `${angkaTampil} seat`}</span>
            {canEdit && (
              <button type="button" className={linkClass} onClick={bukaTampil}>
                Ubah
              </button>
            )}
          </span>
        </div>
      </div>

      {overview.adjustments.length > 0 && (
        <details className="text-xs text-neutral-600">
          <summary className="cursor-pointer font-medium text-neutral-700">Riwayat penyesuaian ({overview.adjustments.length})</summary>
          <ul className="mt-2 space-y-2">
            {overview.adjustments.map(a => (
              <li key={a.id} className="p-2 bg-neutral-50 border border-neutral-200 rounded-md">
                <div className="flex justify-between gap-2">
                  <span className="font-semibold text-neutral-900">
                    {a.jenis === 'tampil'
                      ? `Seat tampil: ${teksTampil(a.seat_before)} → ${teksTampil(a.seat_after)}`
                      : `Sisa seat: ${a.seat_before} → ${a.seat_after}`}
                  </span>
                  <span className="text-neutral-500">{formatWaktu(a.created_at)}</span>
                </div>
                {a.jenis !== 'tampil' && <div className="mt-0.5">{a.reason}</div>}
                <div className="mt-0.5 text-neutral-500">oleh {a.admin_email || 'akun yang sudah dihapus'}</div>
              </li>
            ))}
          </ul>
        </details>
      )}

      <Modal
        isOpen={isTampilOpen}
        onClose={() => !isSavingTampil && setIsTampilOpen(false)}
        title="Seat Tampil"
        size="sm"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setIsTampilOpen(false)} disabled={isSavingTampil}>Batal</Button>
            <Button variant="primary" onClick={() => simpanTampil()} isLoading={isSavingTampil}>Simpan</Button>
          </div>
        }
      >
        <div className="space-y-4">
          {tampilError && <Alert variant="error">{tampilError}</Alert>}
          <Input
            label="Jumlah seat"
            name="seat_tampil"
            type="number"
            min="1"
            max={overview.seat_total}
            value={nilaiTampil}
            onChange={(e) => setNilaiTampil(e.target.value)}
            placeholder="10"
          />
          {overview.seat_tampil != null && (
            <button type="button" className={`text-xs ${linkClass}`} onClick={() => simpanTampil(true)} disabled={isSavingTampil}>
              Matikan
            </button>
          )}
        </div>
      </Modal>
    </div>
  );
};

export default SeatManagement;
