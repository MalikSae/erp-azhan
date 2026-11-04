import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  cariJamaahKaitan,
  gantiKaitanAgen,
  listKaitanLog,
  cariAkunKaitan,
  gantiAgenAkun,
  listKaitanAkunLog,
  listBrands,
  KaitanAgenPicker,
  EMPTY_KAITAN,
  validateKaitan,
} from 'shared';
import { History, Search, Users, Link2, UserRound, IdCard } from 'lucide-react';
import Alert from 'shared/ui/Alert';
import Button from 'shared/ui/Button';
import CustomDropdown from 'shared/ui/CustomDropdown';
import Input from 'shared/ui/Input';
import MetaBox from 'shared/ui/MetaBox';
import PageHeader from 'shared/ui/PageHeader';
import Textarea from 'shared/ui/Textarea';

const SUMBER = {
  jalur1: 'Booking lewat agen',
  jalur2: 'Link referral',
  jalur3: 'Input admin',
  ikut_pic: 'Ikut agen pemesan',
};
const dateTime = (v) => (v ? new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(v)) : '-');

const jamaahLabel = (j) => {
  if (j.kaitan_status === 'terikat_agen') return `Terkunci ke ${j.agen_nama || `agen #${j.agen_id}`}`;
  if (j.kaitan_status === 'tanpa_agen') return 'Tanpa agen';
  return 'Belum ditentukan';
};

const STATUS_AGEN = { tidak_aktif: 'Bukan agen', pengajuan: 'Mengajukan agen', aktif: 'Agen aktif', nonaktif: 'Agen nonaktif' };

// Screen C4 — override admin (spek keagenan 4.4), Admin Master saja. Dua objek:
// jamaah (pindahkan kunci, termasuk komisi yang sudah tercatat) dan akun (pasang/ganti/lepas upline).
const TABS = [
  { key: 'jamaah', label: 'Kunci jamaah', icon: IdCard },
  { key: 'akun', label: 'Agen akun', icon: UserRound },
];

export default function GantiKaitanAgenPage() {
  // Dari form edit jamaah: ?q=<ID jamaah/nama>&jamaah=<id> langsung membuka tab jamaah dan memilih jamaah itu.
  const [params] = useSearchParams();
  const [tab, setTab] = useState(() => (params.get('akun') ? 'akun' : 'jamaah'));
  const [brands, setBrands] = useState([]);
  const [brandId, setBrandId] = useState('');
  const [q, setQ] = useState(() => params.get('q') || '');
  const [autoPilihId, setAutoPilihId] = useState(() => Number(params.get(params.get('akun') ? 'akun' : 'jamaah')) || null);
  const [hasil, setHasil] = useState([]);
  const [mencari, setMencari] = useState(false);
  const [selected, setSelected] = useState(null);
  const [kaitan, setKaitan] = useState(EMPTY_KAITAN);
  const [alasan, setAlasan] = useState('');
  const [pindahkanKomisi, setPindahkanKomisi] = useState(false);
  const [log, setLog] = useState([]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [sukses, setSukses] = useState('');

  const isJamaah = tab === 'jamaah';
  const idOf = (row) => (isJamaah ? row.jamaah_id : row.akun_id);

  useEffect(() => {
    listBrands().then((b) => setBrands(b || [])).catch(() => {});
  }, []);

  const resetForm = () => {
    setKaitan(EMPTY_KAITAN);
    setAlasan('');
    setPindahkanKomisi(false);
  };

  const ganti = (key) => {
    if (key === tab) return;
    setTab(key);
    setHasil([]);
    setSelected(null);
    setLog([]);
    setError('');
    setSukses('');
    setAutoPilihId(null);
    resetForm();
  };

  const loadLog = async (row) => {
    try {
      setLog(isJamaah ? await listKaitanLog(row.jamaah_id) : await listKaitanAkunLog(row.akun_id));
    } catch {
      setLog([]);
    }
  };

  const pilih = async (row) => {
    setSelected(row);
    resetForm();
    setError('');
    setSukses('');
    await loadLog(row);
  };

  const cari = async (keepSelectedId) => {
    if (q.trim().length < 3) {
      setHasil([]);
      return;
    }
    setMencari(true);
    try {
      const items = isJamaah ? await cariJamaahKaitan(q.trim(), brandId) : await cariAkunKaitan(q.trim(), brandId);
      setHasil(items);
      if (keepSelectedId) setSelected(items.find((it) => idOf(it) === keepSelectedId) || null);
      if (autoPilihId) {
        const target = items.find((it) => idOf(it) === autoPilihId);
        setAutoPilihId(null);
        if (target) pilih(target);
      }
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
  }, [q, brandId, tab]);

  // Agen yang sedang mengikat objek terpilih, disembunyikan dari picker karena memilihnya tidak mengubah apa pun.
  const agenSaatIni = selected ? selected.agen_id : undefined;
  const sudahPunyaKomisi = isJamaah && selected?.punya_komisi;
  const akunJadiAgen = !isJamaah && (selected?.status_agen === 'aktif' || selected?.status_agen === 'nonaktif');

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
    if (sudahPunyaKomisi && kaitan.mode === 'agen' && !pindahkanKomisi) {
      setError('Jamaah ini sudah punya komisi tercatat. Centang pindahkan komisi agar komisi ikut ke agen baru.');
      return;
    }
    setSaving(true);
    try {
      if (isJamaah) {
        await gantiKaitanAgen(selected.jamaah_id, {
          mode: kaitan.mode,
          agenAkunId: kaitan.agen?.id,
          alasan: alasan.trim(),
          pindahkanKomisi,
        });
      } else {
        await gantiAgenAkun(selected.akun_id, {
          agenAkunId: kaitan.mode === 'agen' ? kaitan.agen.id : null,
          alasan: alasan.trim(),
        });
      }
      setSukses('Perubahan tersimpan dan tercatat di riwayat.');
      resetForm();
      await loadLog(selected);
      await cari(idOf(selected));
    } catch (err) {
      setError(err.response?.data?.error || 'Perubahan gagal disimpan.');
    } finally {
      setSaving(false);
    }
  };

  const logLabel = (l) => {
    if (isJamaah) {
      return `${l.kaitan_status_lama === 'terikat_agen' ? l.agen_lama_nama : 'Tanpa agen'} → ${l.kaitan_status_baru === 'terikat_agen' ? l.agen_baru_nama : 'Tanpa agen'}`;
    }
    return `${l.agen_lama_nama || 'Tanpa agen'} → ${l.agen_baru_nama || 'Tanpa agen'}`;
  };

  return (
    <div className="space-y-6">
      <PageHeader title="Ganti Kaitan Agen" />

      <div role="tablist" aria-label="Objek override" className="inline-flex rounded-xl border border-neutral-200 bg-white p-1 gap-1">
        {TABS.map(({ key, label, icon: Icon }) => (
          <button
            key={key}
            type="button"
            role="tab"
            aria-selected={tab === key}
            onClick={() => ganti(key)}
            className={`inline-flex items-center gap-2 rounded-lg px-3.5 py-2 text-sm font-semibold transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 ${tab === key ? 'bg-primary-50 text-primary-700' : 'text-neutral-600 hover:bg-neutral-50'}`}
          >
            <Icon size={16} />{label}
          </button>
        ))}
      </div>

      {error && !selected && <Alert variant="error" message={error} />}
      <div className="grid grid-cols-1 lg:grid-cols-5 gap-5 items-start">
        <div className="lg:col-span-2 min-w-0">
          <MetaBox title={isJamaah ? 'Cari Jamaah' : 'Cari Akun'} subtitle={isJamaah ? 'Pilih jamaah yang kuncinya ingin dipindah.' : 'Pilih akun yang upline-nya ingin dipasang, diganti, atau dilepas.'} icon={<Search size={18} />} contentClassName="p-4 sm:p-5 space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-1 gap-3">
              <Input
                label={isJamaah ? 'Cari jamaah' : 'Cari akun'}
                name="cari_kaitan"
                value={q}
                onChange={(e) => setQ(e.target.value)}
                placeholder={isJamaah ? 'Nama, nomor HP, atau ID jamaah' : 'Nama, email, atau nomor WA'}
                className="!mb-0"
              />
              <CustomDropdown
                className="!mb-0"
                label="Brand"
                name="brand_id"
                options={[{ value: '', label: 'Semua brand' }, ...brands.map((b) => ({ value: String(b.id), label: b.name }))]}
                value={brandId}
                onChange={(e) => setBrandId(e?.target ? e.target.value : e)}
                placeholder="Semua brand"
              />
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between text-xs text-neutral-500"><span className="font-semibold text-neutral-700">Hasil pencarian</span><span>{mencari ? 'Mencari...' : `${hasil.length} ${isJamaah ? 'jamaah' : 'akun'}`}</span></div>
              <div className="rounded-xl border border-neutral-200 bg-white divide-y divide-neutral-100 max-h-[28rem] overflow-y-auto" aria-live="polite">
                {mencari && <p className="p-3 text-sm text-neutral-500">Mencari...</p>}
                {!mencari && q.trim().length >= 3 && hasil.length === 0 && <p className="p-3 text-sm text-neutral-500">Tidak ditemukan.</p>}
                {!mencari && q.trim().length < 3 && <p className="p-3 text-sm text-neutral-500">Ketik minimal 3 huruf untuk mencari.</p>}
                {hasil.map((row) => (
                  <button
                    key={idOf(row)}
                    type="button"
                    onClick={() => pilih(row)}
                    aria-pressed={selected ? idOf(selected) === idOf(row) : false}
                    className={`w-full text-left p-4 transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary-500 ${selected && idOf(selected) === idOf(row) ? 'bg-primary-50 border-l-2 border-l-primary-500' : 'hover:bg-neutral-50'}`}
                  >
                    <p className="text-sm font-semibold text-neutral-900 truncate">{row.nama_lengkap}</p>
                    {isJamaah ? (
                      <>
                        <p className="text-xs text-neutral-500 truncate">{row.brand_name} • {row.id_jamaah || '-'}{row.no_hp ? ` • ${row.no_hp}` : ''}</p>
                        <p className="text-xs text-neutral-700 mt-0.5">{jamaahLabel(row)}{row.punya_komisi ? ' • sudah ada komisi' : ''}</p>
                      </>
                    ) : (
                      <>
                        <p className="text-xs text-neutral-500 truncate">{row.brand_name} • {row.email}</p>
                        <p className="text-xs text-neutral-700 mt-0.5">{row.agen_nama ? `Diikat ke ${row.agen_nama}` : 'Tanpa agen'} • {STATUS_AGEN[row.status_agen] || row.status_agen}</p>
                      </>
                    )}
                  </button>
                ))}
              </div>
            </div>
          </MetaBox>
        </div>

        <div className="lg:col-span-3 min-w-0 space-y-5">
          {!selected ? (
            <div className="rounded-2xl border border-neutral-200/80 bg-white shadow-card px-6 py-16 sm:py-24 text-center">
              <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-neutral-50 text-neutral-400"><Users size={26} /></div>
              <h2 className="text-sm font-bold text-neutral-900">{isJamaah ? 'Pilih jamaah terlebih dahulu' : 'Pilih akun terlebih dahulu'}</h2>
              <p className="mx-auto mt-2 max-w-sm text-sm leading-relaxed text-neutral-500">Detail kaitan dan riwayat akan tampil di sini.</p>
              <p className="mx-auto mt-3 max-w-sm text-xs leading-relaxed text-neutral-400">
                {isJamaah
                  ? 'Jamaah terkunci ke agen pertama yang membawanya. Hanya override ini yang bisa memindahkan, untuk semua asal kaitan.'
                  : 'Akun yang dibuat tanpa agen tetap tanpa agen. Hanya override ini yang bisa memasangkannya.'}
              </p>
            </div>
          ) : (
            <>
              <MetaBox
                icon={<Link2 size={18} />}
                title={selected.nama_lengkap}
                subtitle={isJamaah ? `${selected.brand_name} • ${selected.id_jamaah || '-'}` : `${selected.brand_name} • ${selected.email}`}
              >
                <div className="space-y-3 rounded-xl border border-neutral-200/80 bg-neutral-50/70 p-4 text-sm">
                  {isJamaah ? (
                    <>
                      <div className="flex justify-between gap-3"><span className="text-neutral-500">Kunci saat ini</span><span className="font-medium text-neutral-900 text-right">{jamaahLabel(selected)}</span></div>
                      <div className="flex justify-between gap-3"><span className="text-neutral-500">Asal kaitan</span><span className="font-medium text-neutral-900 text-right">{SUMBER[selected.kaitan_sumber] || 'Tidak diketahui'}</span></div>
                      <div className="flex justify-between gap-3"><span className="text-neutral-500">Sudah menghasilkan komisi</span><span className="font-medium text-neutral-900">{selected.punya_komisi ? 'Ya' : 'Belum'}</span></div>
                      <div className="flex justify-between gap-3"><span className="text-neutral-500">Booking belum lunas</span><span className="font-medium text-neutral-900">{selected.booking_belum_lunas}</span></div>
                    </>
                  ) : (
                    <>
                      <div className="flex justify-between gap-3"><span className="text-neutral-500">Upline</span><span className="font-medium text-neutral-900 text-right">{selected.agen_nama || 'Tanpa upline'}</span></div>
                      <div className="flex justify-between gap-3"><span className="text-neutral-500">Status akun</span><span className="font-medium text-neutral-900">{STATUS_AGEN[selected.status_agen] || selected.status_agen}</span></div>
                    </>
                  )}
                </div>

                {sukses && <Alert variant="success" message={sukses} />}
                {error && <Alert variant="error" message={error} />}

                <div className="space-y-4 border-t border-neutral-100 pt-5">
                  <div>
                    <h3 className="text-sm font-bold text-neutral-900">{isJamaah ? 'Kunci baru' : 'Upline baru'}</h3>
                    <p className="mt-1 text-xs text-neutral-500">Tentukan pengganti dan tulis alasan perubahan. Tercatat di riwayat.</p>
                  </div>
                  {isJamaah && selected.booking_belum_lunas > 0 && (
                    <Alert
                      variant="warning"
                      message={`Jamaah ini punya ${selected.booking_belum_lunas} booking belum lunas. Komisi booking tersebut dicatat saat lunas, jadi akan diberikan ke agen yang baru (atau tidak ada komisi bila tanpa agen).`}
                    />
                  )}
                  {sudahPunyaKomisi && (
                    <Alert
                      variant="warning"
                      message="Komisi atas jamaah ini sudah tercatat. Memindahkan kunci tanpa memindahkan komisi ditolak sistem. Jamaah yang sudah punya komisi juga tidak bisa dijadikan tanpa agen."
                    />
                  )}
                  <KaitanAgenPicker
                    name={`ganti_kaitan_${tab}`}
                    value={kaitan}
                    onChange={(v) => { setKaitan(v); setError(''); setSukses(''); }}
                    brandId={selected.brand_id}
                    brandRequired
                    excludeAgenId={agenSaatIni ?? undefined}
                    tanpaAgenLabel={isJamaah ? 'Tanpa Agen' : 'Lepas agen (tanpa agen)'}
                    hint={isJamaah ? 'Perubahan tidak mengubah kunci jamaah lain dalam booking yang sama.' : 'Kunci jamaah yang sudah ada tidak berubah.'}
                  />
                  {sudahPunyaKomisi && kaitan.mode === 'agen' && (
                    <label className="flex items-start gap-2.5 rounded-xl border border-neutral-200 bg-neutral-50 p-3 text-xs text-neutral-800 cursor-pointer">
                      <input type="checkbox" checked={pindahkanKomisi} onChange={(e) => setPindahkanKomisi(e.target.checked)} className="mt-0.5 w-4 h-4 text-primary-500 border-neutral-300 rounded focus:ring-primary-500" />
                      <span><strong className="block text-neutral-900">Pindahkan komisi yang sudah tercatat</strong>Komisi langsung pindah ke agen baru, bonus pembinaan ke upline-nya. Ditolak bila saldo agen lama sudah dicairkan.</span>
                    </label>
                  )}
                  {akunJadiAgen && <p className="text-xs text-neutral-600">Akun ini sudah menjadi agen. Upline baru juga menerima bonus pembinaan dari komisi berikutnya. Komisi yang sudah tercatat tidak berubah.</p>}
                  <Textarea
                    label="Alasan penggantian *"
                    value={alasan}
                    onChange={(e) => { setAlasan(e.target.value); setSukses(''); }}
                    rows={3}
                    placeholder="Contoh: jamaah dibawa agen lain secara langsung, kunci salah agen"
                  />
                  <div className="flex justify-end border-t border-neutral-100 pt-4">
                    <Button variant="primary" onClick={submit} isLoading={saving} className="h-11 w-full sm:w-auto">Simpan Perubahan</Button>
                  </div>
                </div>
              </MetaBox>

              <MetaBox title="Riwayat Penggantian" icon={<History size={18} className="text-neutral-700" />}>
                {log.length === 0 ? (
                  <p className="text-sm text-neutral-500">Belum pernah diganti.</p>
                ) : (
                  <ol className="relative ml-1.5 border-l border-neutral-200 space-y-5">
                    {log.map((l) => (
                      <li key={l.id} className="pl-5 relative">
                        <span className="absolute -left-[5px] top-1.5 w-2.5 h-2.5 rounded-full bg-primary-500 ring-4 ring-white" />
                        <p className="text-sm font-heading font-semibold text-neutral-900">{logLabel(l)}</p>
                        <p className="mt-0.5 text-sm font-body text-neutral-700">{l.alasan}</p>
                        {isJamaah && (l.komisi_dipindah > 0 || l.komisi_dihapus > 0) && (
                          <p className="mt-0.5 text-xs font-body text-neutral-600">Komisi dipindah: {l.komisi_dipindah}{l.komisi_dihapus > 0 ? ` • dihapus: ${l.komisi_dihapus}` : ''}</p>
                        )}
                        {!isJamaah && l.upline_diubah && <p className="mt-0.5 text-xs font-body text-neutral-600">Upline ikut diganti</p>}
                        <p className="mt-0.5 text-[11px] font-body text-neutral-400">{l.diganti_oleh} · {dateTime(l.diganti_at)}</p>
                      </li>
                    ))}
                  </ol>
                )}
              </MetaBox>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
