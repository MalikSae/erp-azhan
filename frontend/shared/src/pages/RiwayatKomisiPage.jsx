import { useEffect, useMemo, useState } from 'react';
import { SlidersHorizontal } from 'lucide-react';
import { useSearchParams } from 'react-router-dom';
import { listAgen, listRiwayatKomisi } from '../api/agen';
import { listBrands } from '../api/brands';
import Alert from '../components/ui/Alert';
import Badge from '../components/ui/Badge';
import Button from '../components/ui/Button';
import CustomDropdown from '../components/ui/CustomDropdown';
import Input from '../components/ui/Input';

import DataTable from '../components/ui/DataTable';

import PageHeader from '../components/ui/PageHeader';
import { dateLabel, JENIS_KOMISI, KETERSEDIAAN, money } from '../utils/agen';

const LIMIT = 200;
const valueOf = (v) => (v?.target ? v.target.value : v);

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
    listAgen().then(setAgenList).catch(() => {});
    if (showBrandColumn) listBrands().then((b) => setBrands(b || [])).catch(() => {});
  }, [showBrandColumn]);

  const agenOptions = filter.brand_id ? agenList.filter((a) => String(a.brand_id) === filter.brand_id) : agenList;

  const filterKey = params.toString();
  useEffect(() => {
    setLoading(true);
    setError('');
    listRiwayatKomisi({ ...filter, limit: LIMIT })
      .then(setItems)
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

  const total = useMemo(() => items.reduce((sum, it) => sum + Number(it.nominal || 0), 0), [items]);

  const columns = [
    { header: 'Tanggal', key: 'created_at', sortable: true },
    { header: 'Agen / Penerima', key: 'penerima_nama', sortable: true },
    { header: 'Jenis', key: 'jenis', sortable: true },
    { header: 'Sumber Jamaah', key: 'sumber_nama', sortable: true },
    { header: 'Booking', key: 'booking_code', sortable: true },
    { header: 'Nominal', key: 'nominal', sortable: true },
    { header: 'Ketersediaan', key: 'ketersediaan' },
  ];

  const renderCell = (row, key) => {
    if (key === 'created_at') return dateLabel(row.created_at);
    if (key === 'penerima_nama') return <span className="font-semibold text-neutral-900">{row.penerima_nama}</span>;
    if (key === 'jenis') return JENIS_KOMISI[row.jenis] || row.jenis;
    if (key === 'booking_code') return <span className="font-mono">{row.booking_code}</span>;
    if (key === 'nominal') return <span className="font-semibold text-neutral-900">{money(row.nominal)}</span>;
    if (key === 'ketersediaan') {
      const meta = KETERSEDIAAN[row.ketersediaan] || { label: row.ketersediaan, variant: 'neutral' };
      return (
        <div>
          <Badge variant={meta.variant}>{meta.label}</Badge>
          {row.ketersediaan === 'tertahan' && (
            <p className="mt-1 text-[11px] text-neutral-500">sampai berangkat {dateLabel(row.berangkat_tanggal)}</p>
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
      <Button type="button" variant={panelOpen || panelActive ? 'dark' : 'secondary'}
        onClick={() => setPanelOpen((open) => !open)} className="h-11 shrink-0 whitespace-nowrap" aria-expanded={panelOpen}>
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
    <div className="space-y-5">
      <PageHeader
        title="Riwayat Komisi"
        subtitle={`Seluruh transaksi komisi agen ${showBrandColumn ? 'lintas brand' : 'di brand ini'}. Hanya baca, tanpa pembatalan.`}
      />
      {error && <Alert variant="error" message={error} />}

      <p className="text-sm text-neutral-600">
        Total {items.length} transaksi: <span className="font-bold text-neutral-900">{money(total)}</span>
        {items.length >= LIMIT && <span className="text-neutral-500"> (menampilkan {LIMIT} terbaru, persempit filter untuk melihat sisanya)</span>}
      </p>

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
