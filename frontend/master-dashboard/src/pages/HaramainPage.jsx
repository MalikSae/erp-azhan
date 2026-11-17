import React, { useCallback, useEffect, useState } from 'react';
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { ArrowDown, ArrowUp } from 'lucide-react';
import PageHeader from 'shared/ui/PageHeader';
import Button from 'shared/ui/Button';
import Badge from 'shared/ui/Badge';
import Alert from 'shared/ui/Alert';
import Modal from 'shared/ui/Modal';
import Table from 'shared/ui/Table';
import SegmentedTabs from 'shared/ui/SegmentedTabs';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import { TableCard } from 'shared/ui/DetailLayout';
import {
  abaikanLaporanUlasan, aturTampilUlasan, deleteHaramain, listHaramain, listUlasanHaramain, urutkanHaramain,
} from '../api/haramain';

export const KATEGORI_HARAMAIN = [
  { value: 'masjid', label: 'Masjid' },
  { value: 'bersejarah', label: 'Tempat Bersejarah' },
  { value: 'ziarah', label: 'Ziarah' },
  { value: 'belanja', label: 'Pusat Belanja' },
  { value: 'kuliner', label: 'Kuliner' },
  { value: 'lainnya', label: 'Lainnya' },
];
export const KOTA_HARAMAIN = [
  { value: 'makkah', label: 'Makkah' },
  { value: 'madinah', label: 'Madinah' },
];
const label = (daftar, v) => daftar.find((x) => x.value === v)?.label || v;
export const mediaSrc = (url) => (url ? `${import.meta.env.VITE_API_BASE_URL || ''}${url}` : null);

const BULAN = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
const waktu = (s) => {
  if (!s) return '';
  const [y, m, d] = s.slice(0, 10).split('-').map(Number);
  return `${d} ${BULAN[m - 1]} ${y}, ${s.slice(11, 16).replace(':', '.')}`;
};

const TABS = [
  { id: 'tempat', label: 'Tempat' },
  { id: 'ulasan', label: 'Ulasan' },
  { id: 'dilaporkan', label: 'Dilaporkan' },
];

// Jelajah Haramain: tempat global (web dan app) dan moderasi ulasan jamaah.
const HaramainPage = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
  const tab = TABS.some((t) => t.id === searchParams.get('tab')) ? searchParams.get('tab') : 'tempat';
  const [tempat, setTempat] = useState(null);
  const [ulasan, setUlasan] = useState(null);
  const [errorMessage, setErrorMessage] = useState(null);
  const [successMessage, setSuccessMessage] = useState(null);
  const [hapus, setHapus] = useState(null);
  const [memproses, setMemproses] = useState(false);

  useEffect(() => {
    if (location.state?.successMessage) {
      setSuccessMessage(location.state.successMessage);
      window.history.replaceState({}, '');
    }
  }, [location.state]);

  const load = useCallback(async () => {
    setErrorMessage(null);
    try {
      if (tab === 'tempat') setTempat(await listHaramain());
      else setUlasan(await listUlasanHaramain(tab === 'dilaporkan' ? 'dilaporkan' : 'semua'));
    } catch (err) {
      setErrorMessage(err.response?.data?.error || 'Gagal memuat data.');
    }
  }, [tab]);

  useEffect(() => {
    setUlasan(null);
    load();
  }, [load]);

  const jalankan = async (fn, pesan) => {
    setErrorMessage(null);
    try {
      await fn();
      if (pesan) setSuccessMessage(pesan);
      await load();
    } catch (err) {
      setErrorMessage(err.response?.data?.error || 'Gagal memproses.');
    }
  };

  const pindah = (index, arah) => {
    const ids = tempat.map((t) => t.id);
    [ids[index], ids[index + arah]] = [ids[index + arah], ids[index]];
    jalankan(() => urutkanHaramain(ids));
  };

  const konfirmasiHapus = async () => {
    setMemproses(true);
    await jalankan(() => deleteHaramain(hapus.id), 'Tempat dihapus.');
    setHapus(null);
    setMemproses(false);
  };

  const kolomTempat = [
    { header: 'Tempat', key: 'nama' },
    { header: 'Kategori', key: 'kategori' },
    { header: 'Kota', key: 'kota' },
    { header: 'Ulasan', key: 'jumlah_ulasan', align: 'right' },
    { header: 'Status', key: 'aktif' },
    { header: '', key: 'aksi', align: 'right' },
  ];
  const selTempat = (t, key) => {
    const i = tempat.findIndex((x) => x.id === t.id);
    if (key === 'nama') {
      return (
        <div className="flex items-center gap-3">
          <img src={mediaSrc(t.foto_url)} alt="" className="h-12 w-16 shrink-0 rounded-md border border-neutral-200 object-cover" />
          <div className="min-w-0">
            <button type="button" onClick={() => navigate(`/haramain/${t.id}/ubah`)} className="text-left font-medium text-neutral-900 hover:text-primary-600">
              {t.nama_lengkap || t.nama}
            </button>
            <div className="text-xs text-neutral-500">{t.ringkas}</div>
          </div>
        </div>
      );
    }
    if (key === 'kategori') return label(KATEGORI_HARAMAIN, t.kategori);
    if (key === 'kota') return label(KOTA_HARAMAIN, t.kota);
    if (key === 'jumlah_ulasan') return <span className="tabular-nums">{t.jumlah_ulasan}</span>;
    if (key === 'aktif') {
      return (
        <div className="flex flex-wrap gap-1">
          {t.aktif ? <Badge variant="success">Aktif</Badge> : <Badge variant="neutral">Nonaktif</Badge>}
          {t.populer && <Badge variant="info">Populer</Badge>}
        </div>
      );
    }
    if (key === 'aksi') {
      return (
        <div className="flex items-center justify-end gap-1">
          <Button size="sm" variant="ghost" aria-label="Naikkan" disabled={i === 0} onClick={() => pindah(i, -1)}><ArrowUp size={16} /></Button>
          <Button size="sm" variant="ghost" aria-label="Turunkan" disabled={i === tempat.length - 1} onClick={() => pindah(i, 1)}><ArrowDown size={16} /></Button>
          <Button size="sm" variant="danger-outline" onClick={() => setHapus(t)}>Hapus</Button>
        </div>
      );
    }
    return t[key];
  };

  const kolomUlasan = [
    { header: 'Ulasan', key: 'teks' },
    { header: 'Tempat', key: 'tempat_nama' },
    { header: 'Laporan', key: 'jumlah_laporan', align: 'right' },
    { header: 'Status', key: 'disembunyikan' },
    { header: '', key: 'aksi', align: 'right' },
  ];
  const selUlasan = (u, key) => {
    if (key === 'teks') {
      return (
        <div className="min-w-0 max-w-md">
          <div className="text-xs text-neutral-500">{u.nama} · {waktu(u.dibuat_at)}</div>
          <p className="mt-0.5 whitespace-pre-line text-sm text-neutral-900">{u.teks}</p>
          {u.foto.length > 0 && (
            <div className="mt-2 flex gap-2">
              {u.foto.map((f) => (
                <a key={f} href={mediaSrc(f)} target="_blank" rel="noreferrer">
                  <img src={mediaSrc(f)} alt="" className="h-12 w-12 rounded-md border border-neutral-200 object-cover" />
                </a>
              ))}
            </div>
          )}
        </div>
      );
    }
    if (key === 'jumlah_laporan') return <span className="tabular-nums">{u.jumlah_laporan}</span>;
    if (key === 'disembunyikan') return u.disembunyikan ? <Badge variant="neutral">Disembunyikan</Badge> : <Badge variant="success">Tayang</Badge>;
    if (key === 'aksi') {
      return (
        <div className="flex items-center justify-end gap-1">
          {tab === 'dilaporkan' && (
            <Button size="sm" variant="secondary" onClick={() => jalankan(() => abaikanLaporanUlasan(u.id), 'Laporan diabaikan.')}>Abaikan</Button>
          )}
          {u.disembunyikan ? (
            <Button size="sm" variant="secondary" onClick={() => jalankan(() => aturTampilUlasan(u.id, false), 'Ulasan ditampilkan.')}>Tampilkan</Button>
          ) : (
            <Button size="sm" variant="danger-outline" onClick={() => jalankan(() => aturTampilUlasan(u.id, true), 'Ulasan disembunyikan.')}>Sembunyikan</Button>
          )}
        </div>
      );
    }
    return u[key];
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Jelajah Haramain"
        actionLabel={tab === 'tempat' ? '+ Tambah Tempat' : undefined}
        onAction={tab === 'tempat' ? () => navigate('/haramain/baru') : undefined}
      />
      {successMessage && <Alert variant="success" onClose={() => setSuccessMessage(null)}>{successMessage}</Alert>}
      {errorMessage && <Alert variant="error" onClose={() => setErrorMessage(null)}>{errorMessage}</Alert>}
      <SegmentedTabs tabs={TABS} value={tab} onChange={(v) => setSearchParams(v === 'tempat' ? {} : { tab: v }, { replace: true })} />

      {tab === 'tempat' ? (
        !tempat ? <LoadingSpinner variant="card" /> : (
          <TableCard title={`Tempat (${tempat.length})`}>
            <div className="overflow-x-auto">
              <Table columns={kolomTempat} data={tempat} renderCell={selTempat} emptyMessage="Belum ada tempat." />
            </div>
          </TableCard>
        )
      ) : (
        !ulasan ? <LoadingSpinner variant="card" /> : (
          <TableCard title={`Ulasan (${ulasan.length})`}>
            <div className="overflow-x-auto">
              <Table columns={kolomUlasan} data={ulasan} renderCell={selUlasan} emptyMessage={tab === 'dilaporkan' ? 'Tidak ada laporan.' : 'Belum ada ulasan.'} />
            </div>
          </TableCard>
        )
      )}

      <Modal
        isOpen={hapus !== null}
        onClose={() => !memproses && setHapus(null)}
        title="Hapus Tempat"
        size="sm"
        footer={(
          <>
            <Button variant="secondary" onClick={() => setHapus(null)} disabled={memproses}>Batal</Button>
            <Button variant="danger" isLoading={memproses} onClick={konfirmasiHapus}>Hapus</Button>
          </>
        )}
      >
        <p className="font-body text-sm text-neutral-700">Hapus {hapus?.nama}? Ulasannya ikut terhapus.</p>
      </Modal>
    </div>
  );
};

export default HaramainPage;
