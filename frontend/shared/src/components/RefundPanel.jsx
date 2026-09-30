import React, { useEffect, useState } from 'react';
import { RotateCcw } from 'lucide-react';
import { listRefunds, createRefund } from '../api/bookings';
import MetaBox from './ui/MetaBox';
import Button from './ui/Button';
import Modal from './ui/Modal';
import Alert from './ui/Alert';
import Input from './ui/Input';
import Textarea from './ui/Textarea';
import CurrencyInput from './ui/CurrencyInput';

const rupiah = (v) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(Number(v) || 0);
const hariIniWIB = () => new Date(Date.now() + 7 * 60 * 60 * 1000).toISOString().slice(0, 10);

// Pengembalian dana booking batal yang sudah dibayar (keputusan audit 2026-09-30).
// Panel hanya tampil untuk booking batal yang punya pembayaran terkonfirmasi.
// Refund hanya dicatat, tidak bisa diubah/dihapus, agar jejak audit utuh.
const RefundPanel = ({ booking, onRecorded }) => {
  const [items, setItems] = useState([]);
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ jumlah: '', tanggal: '', metode: '', catatan: '' });
  const [error, setError] = useState(null);
  const [saving, setSaving] = useState(false);

  const tampil = booking?.status === 'batal' && Number(booking?.total_dibayar) > 0;
  const sisa = Math.max(0, Number(booking?.total_dibayar || 0) - Number(booking?.total_refund || 0));

  useEffect(() => {
    if (!tampil) return undefined;
    let alive = true;
    listRefunds(booking.id)
      .then((data) => alive && setItems(data || []))
      .catch(() => alive && setItems([]));
    return () => { alive = false; };
  }, [booking?.id, booking?.total_refund, tampil]);

  if (!tampil) return null;

  const buka = () => {
    setForm({ jumlah: String(sisa), tanggal: hariIniWIB(), metode: '', catatan: '' });
    setError(null);
    setOpen(true);
  };

  const submit = async (e) => {
    e.preventDefault();
    const jumlah = Math.floor(parseFloat(form.jumlah) || 0);
    if (jumlah <= 0) {
      setError('Jumlah pengembalian harus lebih dari 0');
      return;
    }
    if (jumlah > sisa) {
      setError(`Jumlah maksimal ${rupiah(sisa)} (dana yang belum dikembalikan)`);
      return;
    }
    if (!form.tanggal) {
      setError('Tanggal pengembalian wajib diisi');
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await createRefund(booking.id, { jumlah, tanggal: form.tanggal, metode: form.metode, catatan: form.catatan });
      setOpen(false);
      onRecorded?.();
    } catch (err) {
      setError(err.response?.data?.error || 'Pengembalian dana gagal dicatat');
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      <MetaBox
        title="Pengembalian Dana"
        subtitle="Booking dibatalkan setelah ada pembayaran terkonfirmasi"
        icon={<RotateCcw size={18} className="text-neutral-700" />}
        headerAction={sisa > 0 && (
          <Button size="sm" variant="primary" onClick={buka} className="text-xs">Catat Pengembalian</Button>
        )}
      >
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-sm font-body">
          <div className="rounded-xl border border-neutral-200 bg-neutral-50 p-3">
            <p className="text-xs text-neutral-500">Dana diterima</p>
            <p className="font-semibold text-neutral-900">{rupiah(booking.total_dibayar)}</p>
          </div>
          <div className="rounded-xl border border-neutral-200 bg-neutral-50 p-3">
            <p className="text-xs text-neutral-500">Sudah dikembalikan</p>
            <p className="font-semibold text-neutral-900">{rupiah(booking.total_refund)}</p>
          </div>
          <div className={`rounded-xl border p-3 ${sisa > 0 ? 'border-warning-200 bg-warning-50' : 'border-success-200 bg-success-50'}`}>
            <p className={`text-xs ${sisa > 0 ? 'text-warning-800' : 'text-success-800'}`}>{sisa > 0 ? 'Belum dikembalikan' : 'Selesai'}</p>
            <p className={`font-semibold ${sisa > 0 ? 'text-warning-900' : 'text-success-900'}`}>{rupiah(sisa)}</p>
          </div>
        </div>

        {items.length > 0 && (
          <div className="divide-y divide-neutral-100 pt-2">
            {items.map((rf) => (
              <div key={rf.id} className="py-2.5 flex flex-wrap items-start justify-between gap-2">
                <div>
                  <p className="text-sm font-medium text-neutral-900 font-body">{rupiah(rf.jumlah)}</p>
                  <p className="text-xs text-neutral-500 font-body">
                    {rf.tanggal}{rf.metode ? ` · ${rf.metode}` : ''}
                  </p>
                  {rf.catatan && <p className="text-xs text-neutral-600 font-body">{rf.catatan}</p>}
                </div>
              </div>
            ))}
          </div>
        )}
      </MetaBox>

      <Modal isOpen={open} onClose={() => !saving && setOpen(false)} title="Catat Pengembalian Dana">
        <form onSubmit={submit} className="space-y-4">
          {error && <Alert variant="error">{error}</Alert>}
          <CurrencyInput label="Jumlah Dikembalikan (Rp)" value={form.jumlah} onChange={(v) => setForm((f) => ({ ...f, jumlah: v }))} placeholder="0" />
          <Input label="Tanggal Pengembalian" type="date" value={form.tanggal} onChange={(e) => setForm((f) => ({ ...f, tanggal: e.target.value }))} required />
          <Input label="Metode" value={form.metode} onChange={(e) => setForm((f) => ({ ...f, metode: e.target.value }))} placeholder="Transfer BCA ke rekening jamaah" />
          <Textarea label="Catatan" value={form.catatan} onChange={(e) => setForm((f) => ({ ...f, catatan: e.target.value }))} maxLength={500} placeholder="Nomor referensi transfer atau alasan potongan" />
          <p className="text-xs text-neutral-500 font-body">
            Catatan pengembalian tidak dapat diubah atau dihapus setelah disimpan.
          </p>
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)} disabled={saving}>Batal</Button>
            <Button type="submit" variant="primary" disabled={saving}>{saving ? 'Menyimpan...' : 'Simpan'}</Button>
          </div>
        </form>
      </Modal>
    </>
  );
};

export default RefundPanel;
