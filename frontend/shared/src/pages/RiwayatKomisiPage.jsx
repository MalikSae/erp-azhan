import { useEffect, useMemo, useState } from 'react';
import { SlidersHorizontal, Receipt, Banknote, CheckCircle2, Clock3 } from 'lucide-react';
import { useSearchParams } from 'react-router-dom';
import { listAgen, listRiwayatKomisi } from '../api/agen';
import { listBrands } from '../api/brands';
import Alert from '../components/ui/Alert';
import Button from '../components/ui/Button';
import CustomDropdown from '../components/ui/CustomDropdown';
import Input from '../components/ui/Input';
import DataTable from '../components/ui/DataTable';
import PageHeader from '../components/ui/PageHeader';
import StatTile from '../components/ui/StatTile';
import { dateLabel, JENIS_KOMISI, money } from '../utils/agen';

const LIMIT = 200;
const valueOf = (v) => (v?.target ? v.target.value : v);

const KETERSEDIAAN_STYLE = {
  tersedia: { label: 'Tersedia', text: 'text-success-700', dot: 'bg-success-500' },
  tertahan: { label: 'Tertahan', text: 'text-warning-700', dot: 'bg-warning-500' },
  kredit: { label: 'Kredit cashback', text: 'text-neutral-500', dot: 'bg-neutral-300' },
};

// Screen B3 — ledger transaksi_komisi lintas agen. Read-only, tanpa aksi
// reversal (agen-azhan.md §5.8). showBrandColumn: Admin Master, lintas brand
// dengan filter brand (dikirim ke server sebagai brand_id).
export default function RiwayatKomisiPage({ showBrandColumn = false }) {
  const [params, setParams] = useSearchParams();
  const [panelOpen, setPanelOpen] = useState(false);
  const filter = {
    brand_id: showBrandColumn ? params.get('brand_id') || '' : '',
    agen_id: params.get('agen_id') || '',
    jenis: params.get('jenis') || '',
    dari: params.get('dari') || '',
    sampai: params.get('sampai') || '',
  };
  const [agenList, setAgenList] = useState([]);
  const [brands, setBrands] = useState([]);
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    listAgen().then((data) => setAgenList(data || [])).catch(() => {});
    if (showBrandColumn) listBrands().then((b) => setBrands(b || [])).catch(() => {});
  }, [showBrandColumn]);

  const agenOptions = filter.brand_id ? agenList.filter((a) => String(a.brand_id) === filter.brand_id) : agenList;

  const filterKey = params.toString();
  useEffect(() => {
    setLoading(true);
    setError('');
    listRiwayatKomisi({ ...filter, limit: LIMIT })
      .then((data) => setItems(data || []))
      .catch(() => setError('Riwayat komisi gagal dimuat. Silakan coba kembali.'))
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterKey]);

  const setFilter = (key, value) => {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value);
    else next.delete(key);
    setParams(next, { replace: true });
  };

  const stats = useMemo(() => {
    const sum = (list) => list.reduce((t, it) => t + Number(it.nominal || 0), 0);
    return {
      total: sum(items),
      tersedia: sum(items.filter((it) => it.ketersediaan === 'tersedia')),
      tertahan: sum(items.filter((it) => it.ketersediaan === 'tertahan')),
    };
  }, [items]);

  const columns = [
    { header: 'Tanggal', key: 'created_at', sortable: true },
    { header: 'Penerima', key: 'penerima_nama', sortable: true },
    { header: 'Sumber', key: 'sumber_nama', sortable: true },
    { header: 'Nominal', key: 'nominal', sortable: true, sortFn: (a, b) => Number(a.nominal) - Number(b.nominal) },
    { header: 'Ketersediaan', key: 'ketersediaan' },
  ];

  const renderCell = (row, key) => {
    if (key === 'created_at') return <span className="text-[13px] font-body text-neutral-600 whitespace-nowrap">{dateLabel(row.created_at)}</span>;
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
    if (key === 'nominal') return <span className="text-[13px] font-body font-semibold text-neutral-900 whitespace-nowrap">{money(row.nominal)}</span>;
    if (key === 'ketersediaan') {
      const meta = KETERSEDIAAN_STYLE[row.ketersediaan] || { label: row.ketersediaan, text: 'text-neutral-500', dot: 'bg-neutral-300' };
      return (
        <div>
          <span className={`inline-flex items-center gap-1.5 text-[12px] font-body whitespace-nowrap ${meta.text}`}>
            <span className={`w-2 h-2 rounded-full ${meta.dot}`} />{meta.label}
          </span>
          {row.ketersediaan === 'tertahan' && (
            <p className="mt-0.5 text-[11px] font-body text-neutral-400">sampai berangkat {dateLabel(row.berangkat_tanggal)}</p>
          )}
        </div>
      );
    }
    return row[key] ?? '-';
  };

  const panelActive = ['brand_id', 'agen_id', 'dari', 'sampai'].filter((key) => filter[key]).length;
  const toolbarActions = (
    <div className="flex w-full items-center gap-2 sm:w-auto">
      <CustomDropdown
        value={filter.jenis}
        onChange={(v) => setFilter('jenis', valueOf(v))}
        options={[{ value: '', label: 'Semua jenis' }, ...Object.entries(JENIS_KOMISI).map(([value, label]) => ({ value, label }))]}
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
            onChange={(v) => {
              const next = new URLSearchParams(params);
              const value = valueOf(v);
              if (value) next.set('brand_id', value);
              else next.delete('brand_id');
              next.delete('agen_id');
              setParams(next, { replace: true });
            }}
            options={[{ value: '', label: 'Semua brand' }, ...brands.map((b) => ({ value: String(b.id), label: b.name }))]}
            placeholder="Semua brand" />
        )}
        <CustomDropdown label="Agen" name="agen_id" value={filter.agen_id}
          onChange={(v) => setFilter('agen_id', valueOf(v))}
          options={[{ value: '', label: 'Semua agen' }, ...agenOptions.map((a) => ({
            value: String(a.jamaah_id),
            label: showBrandColumn && !filter.brand_id ? `${a.nama_lengkap} (${a.brand_name})` : a.nama_lengkap,
          }))]} placeholder="Semua agen" />
        <Input label="Dari tanggal" type="date" name="dari" value={filter.dari} onChange={(e) => setFilter('dari', e.target.value)} />
        <Input label="Sampai tanggal" type="date" name="sampai" value={filter.sampai} min={filter.dari || undefined} onChange={(e) => setFilter('sampai', e.target.value)} />
      </div>
      {Object.values(filter).some(Boolean) && (
        <div className="flex justify-end">
          <button type="button" onClick={() => {
            const next = new URLSearchParams(params);
            Object.keys(filter).forEach((key) => next.delete(key));
            setParams(next, { replace: true });
          }} className="text-sm font-semibold text-neutral-900 underline underline-offset-2">Reset semua filter</button>
        </div>
      )}
    </div>
  );
  return (
    <div className="space-y-6">
      <PageHeader title="Riwayat Komisi" />
      {error && <Alert variant="error" message={error} />}

      {!loading && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatTile
            icon={Receipt}
            variant="primary"
            value={items.length}
            label="Transaksi"
            hint={items.length >= LIMIT ? `${LIMIT} terbaru` : undefined}
          />
          <StatTile icon={Banknote} variant="violet" value={money(stats.total)} label="Total komisi" hint="Termasuk cashback" />
          <StatTile icon={CheckCircle2} variant="success" value={money(stats.tersedia)} label="Tersedia" />
          <StatTile icon={Clock3} variant="warning" value={money(stats.tertahan)} label="Tertahan" hint="Sampai berangkat" />
        </div>
      )}

      {items.length >= LIMIT && (
        <p className="text-xs font-body text-neutral-500">Menampilkan {LIMIT} transaksi terbaru. Persempit filter untuk melihat sisanya.</p>
      )}

      <DataTable
        columns={columns}
        data={items}
        renderCell={renderCell}
        itemsPerPage={20}
        toolbarActions={toolbarActions}
        toolbarPanel={toolbarPanel}
        searchPlaceholder="Cari agen, jamaah, atau kode booking..."
        emptyMessage={loading ? 'Memuat riwayat komisi...' : 'Belum ada transaksi komisi'}
      />
    </div>
  );
}
