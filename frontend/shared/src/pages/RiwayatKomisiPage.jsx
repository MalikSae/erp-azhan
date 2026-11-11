import { useEffect, useState } from 'react';
import { SlidersHorizontal, Receipt, Banknote, CheckCircle2, Clock3 } from 'lucide-react';
import { useSearchParams } from 'react-router-dom';
import { listAgen, listRiwayatKomisi } from '../api/agen';
import { listBrandOptions } from '../api/brands';
import Alert from '../components/ui/Alert';
import Button from '../components/ui/Button';
import CustomDropdown from '../components/ui/CustomDropdown';
import Input from '../components/ui/Input';
import DataTable from '../components/ui/DataTable';
import PageHeader from '../components/ui/PageHeader';
import StatTile from '../components/ui/StatTile';
import { dateLabel, HARI_TAHAN_KOMISI, JENIS_KOMISI, JENIS_KOMISI_FILTER, money, SEBAB_BATAL_KOMISI } from '../utils/agen';

const PER_PAGE = 20;
const valueOf = (v) => (v?.target ? v.target.value : v);

const KETERSEDIAAN_STYLE = {
  tersedia: { label: 'Tersedia', text: 'text-success-700', dot: 'bg-success-500' },
  tertahan: { label: 'Tertahan', text: 'text-warning-700', dot: 'bg-warning-500' },
  kredit: { label: 'Kredit cashback', text: 'text-neutral-500', dot: 'bg-neutral-300' },
  batal: { label: 'Dibatalkan', text: 'text-neutral-500', dot: 'bg-neutral-400' },
};

const STATUS_OPTIONS = [
  { value: '', label: 'Komisi aktif' },
  { value: 'batal', label: 'Dibatalkan' },
  { value: 'semua', label: 'Semua' },
];

// Screen B3: ledger transaksi_komisi lintas agen, hanya baca. Paginasi, pencarian, dan ringkasan dihitung
// server, jadi angka di kartu mencakup seluruh hasil filter, bukan hanya halaman yang tampil.
// Komisi mengikuti lunas terakhir: baris yang batal (booking turun dari lunas) tidak dihitung dan hanya
// tampil lewat filter status. showBrandColumn: pengguna holding, dengan filter brand.
export default function RiwayatKomisiPage({ showBrandColumn = false }) {
  const [params, setParams] = useSearchParams();
  const [panelOpen, setPanelOpen] = useState(false);
  const filter = {
    brand_id: showBrandColumn ? params.get('brand_id') || '' : '',
    agen_id: params.get('agen_id') || '',
    jenis: params.get('jenis') || '',
    status: params.get('status') || '',
    dari: params.get('dari') || '',
    sampai: params.get('sampai') || '',
  };
  const [agenList, setAgenList] = useState([]);
  const [brands, setBrands] = useState([]);
  const [hasil, setHasil] = useState({ items: [], ringkasan: { jumlah: 0, total: 0, matang: 0, belum_matang: 0 } });
  const [page, setPage] = useState(1);
  const [input, setInput] = useState('');
  const [q, setQ] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    listAgen().then((data) => setAgenList(data || [])).catch(() => {});
    if (showBrandColumn) listBrandOptions().then((b) => setBrands(b || [])).catch(() => {});
  }, [showBrandColumn]);

  // Pencarian dikirim ke server setelah pengguna berhenti mengetik.
  useEffect(() => {
    const timer = setTimeout(() => { setQ(input.trim()); setPage(1); }, 350);
    return () => clearTimeout(timer);
  }, [input]);

  const agenOptions = filter.brand_id ? agenList.filter((a) => String(a.brand_id) === filter.brand_id) : agenList;

  const filterKey = params.toString();
  useEffect(() => {
    let batal = false;
    setLoading(true);
    setError('');
    listRiwayatKomisi({ ...filter, q, limit: PER_PAGE, offset: (page - 1) * PER_PAGE })
      .then((data) => { if (!batal) setHasil(data); })
      .catch(() => { if (!batal) setError('Riwayat komisi gagal dimuat. Silakan coba kembali.'); })
      .finally(() => { if (!batal) setLoading(false); });
    return () => { batal = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterKey, q, page]);

  const ubahParams = (ubah) => {
    const next = new URLSearchParams(params);
    ubah(next);
    setParams(next, { replace: true });
    setPage(1);
  };
  const setFilter = (key, value) => ubahParams((next) => {
    if (value) next.set(key, value);
    else next.delete(key);
  });

  const columns = [
    { header: 'Tanggal', key: 'created_at' },
    { header: 'Penerima', key: 'penerima_nama' },
    { header: 'Sumber', key: 'sumber_nama' },
    { header: 'Nominal', key: 'nominal' },
    { header: 'Ketersediaan', key: 'ketersediaan' },
  ];

  const renderCell = (row, key) => {
    const batal = row.ketersediaan === 'batal';
    if (key === 'created_at') return <span className="text-[13px] font-body text-neutral-600">{dateLabel(row.created_at)}</span>;
    if (key === 'penerima_nama') {
      return (
        <div className="min-w-0">
          <div className="font-heading font-semibold text-neutral-900 leading-snug">{row.penerima_nama}</div>
          <div className="text-[11px] font-body text-neutral-400">{JENIS_KOMISI[row.jenis] || row.jenis}</div>
        </div>
      );
    }
    if (key === 'sumber_nama') {
      return (
        <div className="min-w-0">
          <div className="text-[13px] font-body text-neutral-800">{row.sumber_nama || '-'}</div>
          <div className="text-[11px] font-body text-neutral-400 font-mono">{row.booking_code}</div>
        </div>
      );
    }
    if (key === 'nominal') {
      return <span className={`text-[13px] font-body font-semibold ${batal ? 'text-neutral-400 line-through' : 'text-neutral-900'}`}>{money(row.nominal)}</span>;
    }
    if (key === 'ketersediaan') {
      const meta = KETERSEDIAAN_STYLE[row.ketersediaan] || { label: row.ketersediaan, text: 'text-neutral-500', dot: 'bg-neutral-300' };
      return (
        <div>
          <span className={`inline-flex items-center gap-1.5 text-[12px] font-body ${meta.text}`}>
            <span className={`w-2 h-2 rounded-full ${meta.dot}`} />{meta.label}
          </span>
          {row.ketersediaan === 'tertahan' && (
            <p className="mt-0.5 text-[11px] font-body text-neutral-400">tersedia mulai {dateLabel(row.tersedia_mulai)}</p>
          )}
          {batal && (
            <p className="mt-0.5 text-[11px] font-body text-neutral-400">
              {SEBAB_BATAL_KOMISI[row.sebab_batal] || 'Booking turun dari lunas'}, {dateLabel(row.dibatalkan_at)}
            </p>
          )}
        </div>
      );
    }
    return row[key] ?? '-';
  };

  const panelActive = ['brand_id', 'agen_id', 'dari', 'sampai'].filter((key) => filter[key]).length;
  const toolbarActions = (
    <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto">
      <CustomDropdown
        value={filter.status}
        onChange={(v) => setFilter('status', valueOf(v))}
        options={STATUS_OPTIONS}
        placeholder="Komisi aktif"
        className="!mb-0 flex-1 sm:w-40 sm:flex-none"
      />
      <CustomDropdown
        value={filter.jenis}
        onChange={(v) => setFilter('jenis', valueOf(v))}
        options={[{ value: '', label: 'Semua jenis' }, ...Object.entries(JENIS_KOMISI_FILTER).map(([value, label]) => ({ value, label }))]}
        placeholder="Semua jenis"
        className="!mb-0 flex-1 sm:w-44 sm:flex-none"
      />
      <Button
        type="button"
        variant="secondary"
        onClick={() => setPanelOpen((open) => !open)}
        className={`h-10 shrink-0 whitespace-nowrap ${panelOpen || panelActive ? '!border-primary-200 !bg-primary-50 !text-primary-700' : ''}`}
        aria-expanded={panelOpen}
      >
        <SlidersHorizontal className="w-4 h-4" />
        <span>Filter{panelActive ? ` (${panelActive})` : ''}</span>
      </Button>
    </div>
  );
  const toolbarPanel = panelOpen && (
    <div className="space-y-3">
      <div className={`grid grid-cols-1 gap-3 sm:grid-cols-2 ${showBrandColumn ? 'lg:grid-cols-4' : 'lg:grid-cols-3'}`}>
        {showBrandColumn && (
          <CustomDropdown label="Brand" name="brand_id" value={filter.brand_id}
            onChange={(v) => ubahParams((next) => {
              const value = valueOf(v);
              if (value) next.set('brand_id', value);
              else next.delete('brand_id');
              next.delete('agen_id');
            })}
            options={[{ value: '', label: 'Semua brand' }, ...brands.map((b) => ({ value: String(b.id), label: b.name }))]}
            placeholder="Semua brand" />
        )}
        <CustomDropdown label="Agen" name="agen_id" value={filter.agen_id}
          onChange={(v) => setFilter('agen_id', valueOf(v))}
          options={[{ value: '', label: 'Semua agen' }, ...agenOptions.map((a) => ({
            value: String(a.akun_id),
            label: showBrandColumn && !filter.brand_id ? `${a.nama_lengkap} (${a.brand_name})` : a.nama_lengkap,
          }))]} placeholder="Semua agen" />
        <Input label="Dari tanggal" type="date" name="dari" value={filter.dari} onChange={(e) => setFilter('dari', e.target.value)} />
        <Input label="Sampai tanggal" type="date" name="sampai" value={filter.sampai} min={filter.dari || undefined} onChange={(e) => setFilter('sampai', e.target.value)} />
      </div>
      {Object.values(filter).some(Boolean) && (
        <div className="flex justify-end">
          <button type="button" onClick={() => ubahParams((next) => Object.keys(filter).forEach((key) => next.delete(key)))}
            className="text-sm font-semibold text-neutral-900 underline underline-offset-2">Reset semua filter</button>
        </div>
      )}
    </div>
  );

  const r = hasil.ringkasan || {};
  return (
    <div className="space-y-6">
      <PageHeader title="Riwayat Komisi" />
      {error && <Alert variant="error" message={error} />}

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatTile icon={Receipt} variant="primary" value={r.jumlah ?? 0} label="Transaksi" hint="Sesuai filter" />
        <StatTile icon={Banknote} variant="violet" value={money(r.total)} label="Total komisi" hint="Komisi aktif" />
        <StatTile icon={CheckCircle2} variant="success" value={money(r.matang)} label="Sudah matang" hint="Belum dikurangi pencairan" />
        <StatTile icon={Clock3} variant="warning" value={money(r.belum_matang)} label="Tertahan" hint={`Sampai H+${HARI_TAHAN_KOMISI} berangkat`} />
      </div>

      <DataTable
        columns={columns}
        data={hasil.items || []}
        renderCell={renderCell}
        toolbarActions={toolbarActions}
        toolbarPanel={toolbarPanel}
        searchPlaceholder="Cari agen, jamaah, atau kode booking..."
        emptyMessage={loading ? 'Memuat riwayat komisi...' : 'Belum ada transaksi komisi'}
        server={{ total: r.jumlah ?? 0, page, perPage: PER_PAGE, onPage: setPage, query: input, onQuery: setInput }}
      />
    </div>
  );
}
