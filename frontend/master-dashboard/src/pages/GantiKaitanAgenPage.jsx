import { useEffect, useState } from 'react';
import {
  cariJamaahKaitan,
  gantiKaitanAgen,
  listKaitanLog,
  listBrands,
  KaitanAgenPicker,
  EMPTY_KAITAN,
  validateKaitan,
} from 'shared';
import { Search, Lock, History } from 'lucide-react';
import Alert from '../components/ui/Alert';
import Badge from '../components/ui/Badge';
import Button from '../components/ui/Button';
import CustomDropdown from '../components/ui/CustomDropdown';
import MetaBox from '../components/ui/MetaBox';
import PageHeader from '../components/ui/PageHeader';
import Textarea from '../components/ui/Textarea';

const SUMBER = {
  jalur1: 'Jalur 1 (booking agen)',
  jalur2: 'Jalur 2 (link referral)',
  jalur3: 'Jalur 3 (input Admin)',
  ikut_pic: 'Ikut agen PIC',
};
const dateTime = (v) => (v ? new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(v)) : '-');

const kaitanLabel = (j) => {
  if (j.kaitan_status === 'terikat_agen') return `Terikat ke ${j.agen_nama || `agen #${j.agen_id}`}`;
  if (j.kaitan_status === 'tanpa_agen') return 'Tanpa agen';
  return 'Belum ditentukan';
};

// Screen C4 — ganti kaitan agen hasil Jalur 3 (Admin Master saja). Ditolak
// sistem bila jamaah sudah pernah menghasilkan komisi (agen-azhan.md 7.8).
export default function GantiKaitanAgenPage() {
  const [brands, setBrands] = useState([]);
  const [brandId, setBrandId] = useState('');
  const [q, setQ] = useState('');
  const [hasil, setHasil] = useState([]);
  const [mencari, setMencari] = useState(false);
  const [selected, setSelected] = useState(null);
  const [kaitan, setKaitan] = useState(EMPTY_KAITAN);
  const [alasan, setAlasan] = useState('');
  const [log, setLog] = useState([]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [sukses, setSukses] = useState('');

  useEffect(() => {
    listBrands().then((b) => setBrands(b || [])).catch(() => {});
  }, []);

  const cari = async (keepSelectedId) => {
    if (q.trim().length < 3) {
      setHasil([]);
      return;
    }
    setMencari(true);
    try {
      const items = await cariJamaahKaitan(q.trim(), brandId);
      setHasil(items);
      if (keepSelectedId) setSelected(items.find((it) => it.jamaah_id === keepSelectedId) || null);
    } catch {
      setError('Pencarian gagal. Silakan coba kembali.');
    } finally {
      setMencari(false);
    }
  };

  useEffect(() => {
    const timer = setTimeout(() => cari(), 350);
    return () => clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q, brandId]);

  const pilih = async (j) => {
    setSelected(j);
    setKaitan(EMPTY_KAITAN);
    setAlasan('');
    setError('');
    setSukses('');
    try {
      setLog(await listKaitanLog(j.jamaah_id));
    } catch {
      setLog([]);
    }
  };

  const submit = async () => {
    setError('');
    setSukses('');
    const kErr = validateKaitan(kaitan);
    if (kErr) {
      setError(kErr);
      return;
    }
    if (alasan.trim().length < 5) {
      setError('Alasan wajib diisi (minimal 5 karakter).');
      return;
    }
    setSaving(true);
    try {
      await gantiKaitanAgen(selected.jamaah_id, { mode: kaitan.mode, agenJamaahId: kaitan.agen?.id, alasan: alasan.trim() });
      setSukses('Kaitan agen berhasil diganti dan tercatat di log.');
      setKaitan(EMPTY_KAITAN);
      setAlasan('');
      setLog(await listKaitanLog(selected.jamaah_id));
      await cari(selected.jamaah_id);
    } catch (err) {
      setError(err.response?.data?.error || 'Kaitan agen gagal diganti.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-5">
      <PageHeader
        title="Ganti Kaitan Agen"
        subtitle="Koreksi kaitan agen yang salah input (Jalur 3). Hanya bisa selama jamaah belum pernah menghasilkan komisi."
      />

      <div className="grid grid-cols-1 lg:grid-cols-5 gap-5 items-start">
        <div className="lg:col-span-2 space-y-3">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-1 gap-3">
            <div className="flex flex-col gap-1.5">
              <label className="text-sm font-semibold text-neutral-900 font-heading">Cari jamaah</label>
              <div className="relative">
                <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-neutral-400" />
                <input
                  type="text"
                  value={q}
                  onChange={(e) => setQ(e.target.value)}
                  placeholder="Nama, no HP, atau ID jamaah (min. 3 huruf)"
                  className="h-11 w-full min-w-0 rounded-xl border border-neutral-200/90 bg-white pl-9 pr-3.5 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-neutral-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40"
                />
              </div>
            </div>
            <CustomDropdown
              label="Brand"
              name="brand_id"
              options={[{ value: '', label: 'Semua brand' }, ...brands.map((b) => ({ value: String(b.id), label: b.name }))]}
              value={brandId}
              onChange={(e) => setBrandId(e?.target ? e.target.value : e)}
              placeholder="Semua brand"
            />
          </div>

          <div className="rounded-xl border border-neutral-200 bg-white divide-y divide-neutral-100 max-h-[480px] overflow-y-auto">
            {mencari && <p className="p-3 text-sm text-neutral-500">Mencari...</p>}
            {!mencari && q.trim().length >= 3 && hasil.length === 0 && <p className="p-3 text-sm text-neutral-500">Jamaah tidak ditemukan.</p>}
            {!mencari && q.trim().length < 3 && <p className="p-3 text-sm text-neutral-500">Ketik minimal 3 huruf untuk mencari.</p>}
            {hasil.map((j) => (
              <button
                key={j.jamaah_id}
                type="button"
                onClick={() => pilih(j)}
                className={`w-full text-left p-3 hover:bg-neutral-50 transition-colors cursor-pointer ${selected?.jamaah_id === j.jamaah_id ? 'bg-neutral-50' : ''}`}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="text-sm font-semibold text-neutral-900 truncate">{j.nama_lengkap}</p>
                    <p className="text-xs text-neutral-500 truncate">{j.brand_name} • {j.id_jamaah || '-'}{j.no_hp ? ` • ${j.no_hp}` : ''}</p>
                    <p className="text-xs text-neutral-700 mt-0.5">{kaitanLabel(j)}</p>
                  </div>
                  {j.bisa_diganti ? <Badge variant="success">Bisa diganti</Badge> : <Badge variant="neutral" icon={<Lock size={12} />}>Terkunci</Badge>}
                </div>
              </button>
            ))}
          </div>
        </div>

        <div className="lg:col-span-3 space-y-5">
          {!selected ? (
            <div className="rounded-xl border border-dashed border-neutral-300 p-6 text-center text-sm text-neutral-500">
              Pilih jamaah dari hasil pencarian.
            </div>
          ) : (
            <>
              <MetaBox title={selected.nama_lengkap} subtitle={`${selected.brand_name} • ${selected.id_jamaah || '-'}`}>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between gap-3"><span className="text-neutral-500">Kaitan saat ini</span><span className="font-medium text-neutral-900 text-right">{kaitanLabel(selected)}</span></div>
                  <div className="flex justify-between gap-3"><span className="text-neutral-500">Asal kaitan</span><span className="font-medium text-neutral-900 text-right">{SUMBER[selected.kaitan_sumber] || 'Tidak diketahui'}</span></div>
                  <div className="flex justify-between gap-3"><span className="text-neutral-500">Sudah menghasilkan komisi</span><span className="font-medium text-neutral-900">{selected.punya_komisi ? 'Ya' : 'Belum'}</span></div>
                </div>

                {sukses && <Alert variant="success" message={sukses} />}
                {error && <Alert variant="error" message={error} />}

                {selected.bisa_diganti ? (
                  <div className="space-y-4 border-t border-neutral-100 pt-4">
                    <KaitanAgenPicker
                      name="ganti_kaitan"
                      value={kaitan}
                      onChange={(v) => { setKaitan(v); setError(''); }}
                      brandId={selected.brand_id}
                      brandRequired
                      hint="Perubahan tidak mengubah kaitan jamaah lain dalam booking yang sama."
                    />
                    <Textarea
                      label="Alasan penggantian *"
                      value={alasan}
                      onChange={(e) => setAlasan(e.target.value)}
                      rows={3}
                      placeholder="Contoh: salah pilih agen saat input, seharusnya tanpa agen"
                    />
                    <div className="flex justify-end">
                      <Button variant="primary" onClick={submit} isLoading={saving}>Simpan Perubahan</Button>
                    </div>
                  </div>
                ) : (
                  <Alert variant="warning" message={selected.alasan_terkunci} />
                )}
              </MetaBox>

              <MetaBox title="Riwayat Penggantian" icon={<History size={18} className="text-neutral-700" />}>
                {log.length === 0 ? (
                  <p className="text-sm text-neutral-500">Belum pernah diganti.</p>
                ) : (
                  <div className="space-y-3">
                    {log.map((l) => (
                      <div key={l.id} className="rounded-xl border border-neutral-200 p-3 space-y-1 text-sm">
                        <p className="font-medium text-neutral-900">
                          {l.kaitan_status_lama === 'terikat_agen' ? l.agen_lama_nama : 'Tanpa agen'} → {l.kaitan_status_baru === 'terikat_agen' ? l.agen_baru_nama : 'Tanpa agen'}
                        </p>
                        <p className="text-neutral-700">{l.alasan}</p>
                        <p className="text-xs text-neutral-500">{l.diganti_oleh} • {dateTime(l.diganti_at)}</p>
                      </div>
                    ))}
                  </div>
                )}
              </MetaBox>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
