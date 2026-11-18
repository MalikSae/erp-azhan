import React, { useCallback, useContext, useEffect, useState } from 'react';
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { ArrowDown, ArrowUp } from 'lucide-react';
import PageHeader from 'shared/ui/PageHeader';
import Button from 'shared/ui/Button';
import Badge from 'shared/ui/Badge';
import Alert from 'shared/ui/Alert';
import Modal from 'shared/ui/Modal';
import Table from 'shared/ui/Table';
import CustomDropdown from 'shared/ui/CustomDropdown';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import { Surface, Section, TableCard } from 'shared/ui/DetailLayout';
import { AuthContext } from '../context/AuthContext';
import { labelTujuan } from '../components/TujuanPromosiField';
import { deleteBannerPromosi, listBannerPromosi, urutkanBannerPromosi } from '../api/bannerPromosi';
import { listSchedulesAdmin } from '../api/schedules';
import { listBrandOptions } from '../api/brands';

const BULAN = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
const tanggal = (iso) => {
  if (!iso) return '';
  const [y, m, d] = iso.split('-').map(Number);
  return `${d} ${BULAN[m - 1]} ${y}`;
};
const jadwal = (b) => (b.mulai || b.selesai ? `${tanggal(b.mulai) || '…'} – ${tanggal(b.selesai) || '…'}` : 'Tanpa batas');
const STATUS = {
  tayang: { label: 'Tayang', variant: 'success' },
  terjadwal: { label: 'Terjadwal', variant: 'info' },
  berakhir: { label: 'Berakhir', variant: 'neutral' },
  nonaktif: { label: 'Nonaktif', variant: 'neutral' },
};
const valueOf = (v) => (v?.target ? v.target.value : v);
export const mediaSrc = (url) => (url ? `${import.meta.env.VITE_API_BASE_URL || ''}${url}` : null);

// Banner Promosi: carousel Beranda app Hana (maks 5 tayang).
const BannerPromosiPage = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { isHolding } = useContext(AuthContext);
  const [searchParams, setSearchParams] = useSearchParams();
  const brandId = isHolding ? searchParams.get('brand') || '' : '';
  const [brands, setBrands] = useState([]);
  const [data, setData] = useState(null);
  const [paket, setPaket] = useState([]);
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

  useEffect(() => {
    if (isHolding) listBrandOptions().then((b) => setBrands(b || [])).catch(() => {});
  }, [isHolding]);

  const perluBrand = isHolding && !brandId;

  const load = useCallback(async () => {
    if (perluBrand) {
      setData(null);
      return;
    }
    setErrorMessage(null);
    try {
      const [d, s] = await Promise.all([listBannerPromosi(brandId), listSchedulesAdmin().catch(() => [])]);
      setData(d);
      const brandAktif = brandId ? Number(brandId) : null;
      setPaket((s || []).filter((p) => p.status !== 'archived' && (!brandAktif || p.brand_id === brandAktif)));
    } catch (err) {
      setErrorMessage(err.response?.data?.error || 'Gagal memuat banner.');
    }
  }, [brandId, perluBrand]);

  useEffect(() => { load(); }, [load]);

  const pindah = async (index, arah) => {
    const ids = data.banner.map((b) => b.id);
    const tujuan = index + arah;
    [ids[index], ids[tujuan]] = [ids[tujuan], ids[index]];
    setErrorMessage(null);
    try {
      await urutkanBannerPromosi(brandId ? Number(brandId) : null, ids);
      await load();
    } catch (err) {
      setErrorMessage(err.response?.data?.error || 'Gagal mengubah urutan.');
    }
  };

  const konfirmasiHapus = async () => {
    setMemproses(true);
    try {
      await deleteBannerPromosi(hapus.id);
      setHapus(null);
      setSuccessMessage('Banner dihapus.');
      await load();
    } catch (err) {
      setErrorMessage(err.response?.data?.error || 'Gagal menghapus banner.');
      setHapus(null);
    } finally {
      setMemproses(false);
    }
  };

  const kolom = [
    { header: 'Banner', key: 'judul' },
    { header: 'Tujuan', key: 'tujuan' },
    { header: 'Jadwal', key: 'jadwal' },
    { header: 'Status', key: 'status' },
    { header: 'Klik', key: 'klik', align: 'right' },
    { header: '', key: 'aksi', align: 'right' },
  ];
  const sel = (b, key) => {
    const i = data.banner.findIndex((x) => x.id === b.id);
    if (key === 'judul') {
      return (
        <div className="flex items-center gap-3">
          <img src={mediaSrc(b.gambar_url)} alt="" className="h-12 w-24 shrink-0 rounded-md border border-neutral-200 object-cover" />
          <button type="button" onClick={() => navigate(`/banner-promosi/${b.id}/ubah${brandId ? `?brand=${brandId}` : ''}`)} className="text-left font-medium text-neutral-900 hover:text-primary-600">
            {b.judul}
          </button>
        </div>
      );
    }
    if (key === 'tujuan') return <span className="text-neutral-700">{labelTujuan(b.tujuan, b.tujuan_nilai, paket)}</span>;
    if (key === 'jadwal') return <span className="whitespace-nowrap text-neutral-700">{jadwal(b)}</span>;
    if (key === 'status') return <Badge variant={STATUS[b.status]?.variant}>{STATUS[b.status]?.label}</Badge>;
    if (key === 'klik') return <span className="tabular-nums">{b.klik}</span>;
    if (key === 'aksi') {
      return (
        <div className="flex items-center justify-end gap-1">
          <Button size="sm" variant="ghost" aria-label="Naikkan" disabled={i === 0} onClick={() => pindah(i, -1)}><ArrowUp size={16} /></Button>
          <Button size="sm" variant="ghost" aria-label="Turunkan" disabled={i === data.banner.length - 1} onClick={() => pindah(i, 1)}><ArrowDown size={16} /></Button>
          <Button size="sm" variant="danger-outline" onClick={() => setHapus(b)}>Hapus</Button>
        </div>
      );
    }
    return b[key];
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Banner Promosi"
        actionLabel={perluBrand ? undefined : '+ Tambah Banner'}
        onAction={perluBrand ? undefined : () => navigate(`/banner-promosi/baru${brandId ? `?brand=${brandId}` : ''}`)}
      >
        {isHolding && (
          <CustomDropdown
            value={brandId}
            onChange={(v) => setSearchParams(valueOf(v) ? { brand: valueOf(v) } : {}, { replace: true })}
            className="!mb-0 w-48"
            options={[{ value: '', label: 'Pilih brand' }, ...brands.map((b) => ({ value: String(b.id), label: b.name }))]}
          />
        )}
      </PageHeader>
      {successMessage && <Alert variant="success" onClose={() => setSuccessMessage(null)}>{successMessage}</Alert>}
      {errorMessage && <Alert variant="error" onClose={() => setErrorMessage(null)}>{errorMessage}</Alert>}

      {perluBrand ? (
        <Surface><Section title="Pilih brand"><p className="font-body text-sm text-neutral-500">Pilih brand.</p></Section></Surface>
      ) : !data ? (
        <LoadingSpinner variant="card" />
      ) : (
        <>
          <TableCard title={`Banner (${data.banner.length})`}>
            <div className="overflow-x-auto">
              <Table columns={kolom} data={data.banner} renderCell={sel} emptyMessage="Belum ada banner." />
            </div>
          </TableCard>
        </>
      )}

      <Modal
        isOpen={hapus !== null}
        onClose={() => !memproses && setHapus(null)}
        title="Hapus Banner"
        size="sm"
        footer={(
          <>
            <Button variant="secondary" onClick={() => setHapus(null)} disabled={memproses}>Batal</Button>
            <Button variant="danger" isLoading={memproses} onClick={konfirmasiHapus}>Hapus</Button>
          </>
        )}
      >
        <p className="font-body text-sm text-neutral-700">Hapus {hapus?.judul}?</p>
      </Modal>
    </div>
  );
};

export default BannerPromosiPage;
