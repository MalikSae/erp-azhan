import React, { useEffect, useState } from 'react';
import { Coins } from 'lucide-react';
import { getKreditCashback, pakaiKreditCashback } from '../api/agen';
import MetaBox from './ui/MetaBox';
import Button from './ui/Button';
import Modal from './ui/Modal';
import Alert from './ui/Alert';
import CurrencyInput from './ui/CurrencyInput';

const rupiah = (v) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(Number(v) || 0);

// Screen B5 (keputusan D4): kredit cashback pax booking ini dipakai sebagai
// potongan tagihan. Panel tidak tampil bila tidak ada pax yang punya kredit.
// refreshKey: nilai yang berubah setiap data booking dimuat ulang.
const KreditCashbackPanel = ({ bookingId, refreshKey, onApplied }) => {
  const [info, setInfo] = useState(null);
  const [selected, setSelected] = useState(null);
  const [nominal, setNominal] = useState('');
  const [error, setError] = useState(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let alive = true;
    getKreditCashback(bookingId)
      .then((data) => alive && setInfo(data))
      .catch(() => alive && setInfo(null));
    return () => { alive = false; };
  }, [bookingId, refreshKey]);

  if (!info || info.pax.length === 0) return null;

  const maks = selected ? Math.min(selected.kredit_cashback, info.sisa_tagihan) : 0;

  const open = (pax) => {
    setSelected(pax);
    setNominal(String(Math.min(pax.kredit_cashback, info.sisa_tagihan)));
    setError(null);
  };

  const submit = async (e) => {
    e.preventDefault();
    const n = Math.floor(parseFloat(nominal) || 0);
    if (n <= 0) {
      setError('Nominal harus lebih dari 0');
      return;
    }
    if (n > maks) {
      setError(`Nominal maksimal ${rupiah(maks)} (kredit tersedia dan sisa tagihan)`);
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await pakaiKreditCashback(bookingId, selected.jamaah_id, n);
      setSelected(null);
      onApplied?.();
    } catch (err) {
      setError(err.response?.data?.error || 'Kredit cashback gagal dipakai');
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      <MetaBox
        title="Kredit Cashback"
        subtitle="Cashback repeat order milik pax, bisa dipakai sebagai potongan tagihan"
        icon={<Coins size={18} className="text-neutral-700" />}
      >
        <div className="divide-y divide-neutral-100">
          {info.pax.map((p) => (
            <div key={p.jamaah_id} className="py-2.5 flex flex-wrap items-center justify-between gap-2">
              <div>
                <p className="text-sm font-medium text-neutral-900 font-body">{p.nama_lengkap}</p>
                <span className="text-xs text-neutral-500 font-body">Kredit tersedia {rupiah(p.kredit_cashback)}</span>
              </div>
              <Button size="sm" variant="secondary" onClick={() => open(p)} disabled={!info.bisa_dipakai} className="text-xs">
                Pakai sebagai Potongan
              </Button>
            </div>
          ))}
        </div>
        {!info.bisa_dipakai && (
          <p className="text-xs text-neutral-500 font-body">Kredit hanya bisa dipakai pada booking berstatus baru atau DP yang masih punya sisa tagihan.</p>
        )}
      </MetaBox>

      <Modal isOpen={Boolean(selected)} onClose={() => !saving && setSelected(null)} title="Pakai Kredit Cashback">
        {selected && (
          <form onSubmit={submit} className="space-y-4">
            {error && <Alert variant="error">{error}</Alert>}
            <div className="rounded-xl border border-neutral-200 bg-neutral-50 p-3 text-sm font-body space-y-1">
              <div className="flex justify-between gap-2"><span className="text-neutral-500">Jamaah</span><span className="font-semibold text-neutral-900">{selected.nama_lengkap}</span></div>
              <div className="flex justify-between gap-2"><span className="text-neutral-500">Kredit tersedia</span><span className="text-neutral-900">{rupiah(selected.kredit_cashback)}</span></div>
              <div className="flex justify-between gap-2"><span className="text-neutral-500">Sisa tagihan</span><span className="text-neutral-900">{rupiah(info.sisa_tagihan)}</span></div>
            </div>
            <CurrencyInput label="Nominal Potongan (Rp)" value={nominal} onChange={setNominal} placeholder="0" />
            <p className="text-xs text-neutral-500 font-body">
              Tercatat sebagai diskon booking. Menghapus diskon ini mengembalikan kredit jamaah.
            </p>
            <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
              <Button type="button" variant="ghost" onClick={() => setSelected(null)} disabled={saving}>Batal</Button>
              <Button type="submit" variant="primary" disabled={saving}>{saving ? 'Menyimpan...' : 'Pakai Kredit'}</Button>
            </div>
          </form>
        )}
      </Modal>
    </>
  );
};

export default KreditCashbackPanel;
