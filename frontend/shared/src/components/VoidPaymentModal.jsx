import React, { useEffect, useState } from 'react';
import { voidPayment } from '../api/bookings';
import Modal from './ui/Modal';
import Button from './ui/Button';
import Alert from './ui/Alert';

const rupiah = (v) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(Number(v) || 0);

const ALASAN_MIN = 5;
const ALASAN_MAKS = 500;

// Pembatalan pembayaran yang sudah dikonfirmasi (audit 10 November 2026, T-03). Dipakai di Detail Booking
// dan layar Pembayaran. payment = null menutup modal; onDone dipanggil setelah pembatalan tersimpan.
const VoidPaymentModal = ({ payment, onClose, onDone }) => {
  const [alasan, setAlasan] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    setAlasan('');
    setError('');
  }, [payment?.id]);

  const panjang = alasan.trim().length;
  const valid = panjang >= ALASAN_MIN && panjang <= ALASAN_MAKS;

  const submit = async (e) => {
    e.preventDefault();
    if (!payment || !valid || saving) return;
    setSaving(true);
    setError('');
    try {
      const hasil = await voidPayment(payment.id, alasan.trim());
      onDone?.(hasil);
    } catch (err) {
      setError(err.response?.data?.error || 'Gagal membatalkan pembayaran');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal isOpen={Boolean(payment)} onClose={saving ? () => {} : onClose} title="Batalkan Pembayaran" size="md">
      {payment && (
        <form onSubmit={submit} className="space-y-4 font-body">
          {error && <Alert variant="error">{error}</Alert>}
          <div className="rounded-xl border border-warning-200 bg-warning-50 px-3.5 py-3 text-xs text-warning-900 leading-relaxed">
            Pembayaran sebesar <span className="font-semibold">{rupiah(payment.jumlah)}</span> tidak lagi dihitung sebagai dana masuk.
            Status booking dihitung ulang dan bisa turun dari Lunas atau DP. Kursi booking tetap terkunci.
            Komisi agen yang sudah tercatat tidak ikut dibatalkan. Tindakan ini tidak dapat dibalik;
            bila keliru, catat ulang pembayarannya.
          </div>
          <div>
            <label htmlFor="alasan-void" className="block text-xs font-semibold text-neutral-700 uppercase tracking-wider mb-1.5">
              Alasan Pembatalan <span className="text-danger-500">*</span>
            </label>
            <textarea
              id="alasan-void"
              rows={3}
              maxLength={ALASAN_MAKS}
              className="w-full rounded-xl border border-neutral-200 bg-white px-3.5 py-2.5 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/20 resize-none disabled:bg-neutral-100"
              placeholder="Contoh: salah ketik nominal, seharusnya Rp 2.500.000"
              value={alasan}
              onChange={(e) => setAlasan(e.target.value)}
              required
              disabled={saving}
            />
            <p className="text-[11px] text-neutral-400 mt-1">
              Minimal {ALASAN_MIN} karakter. Alasan, nama Anda, dan waktu pembatalan tercatat pada pembayaran ini.
            </p>
          </div>
          <div className="flex justify-end gap-3 pt-2">
            <Button type="button" variant="ghost" onClick={onClose} disabled={saving}>Kembali</Button>
            <Button type="submit" variant="danger" disabled={saving || !valid} isLoading={saving}>
              {saving ? 'Membatalkan...' : 'Batalkan Pembayaran'}
            </Button>
          </div>
        </form>
      )}
    </Modal>
  );
};

export default VoidPaymentModal;
