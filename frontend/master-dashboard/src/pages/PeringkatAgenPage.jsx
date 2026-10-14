import { useEffect, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Coins, Trophy, Users } from 'lucide-react';
import { getPeringkatAgen, listBrands, money } from 'shared';
import Alert from 'shared/ui/Alert';
import CustomDropdown from 'shared/ui/CustomDropdown';
import DataTable from 'shared/ui/DataTable';
import Input from 'shared/ui/Input';
import PageHeader from 'shared/ui/PageHeader';
import StatTile from 'shared/ui/StatTile';

const ymd = (d) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;

// Preset periode, dihitung dari tanggal hari ini (zona waktu browser).
const PRESETS = {
  bulan_ini: { label: 'Bulan ini', range: (t) => [new Date(t.getFullYear(), t.getMonth(), 1), t] },
  bulan_lalu: {
    label: 'Bulan lalu',
    range: (t) => [new Date(t.getFullYear(), t.getMonth() - 1, 1), new Date(t.getFullYear(), t.getMonth(), 0)],
  },
  tiga_bulan: { label: '3 bulan terakhir', range: (t) => [new Date(t.getFullYear(), t.getMonth() - 2, 1), t] },
  tahun_ini: { label: 'Tahun ini', range: (t) => [new Date(t.getFullYear(), 0, 1), t] },
};

const presetRange = (key) => PRESETS[key].range(new Date()).map(ymd);

const TOP = 3;

// Chip brand: warna brand hanya penanda (dot), bukan warna UI.
const BrandChip = ({ name, color }) => (
  <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[12px] font-medium font-body bg-white border border-neutral-200 text-neutral-700 whitespace-nowrap">
    <span className="w-2 h-2 rounded-full shrink-0" style={{ backgroundColor: color || '#9CA3AF' }} />
    {name}
  </span>
);

// Peringkat agen per periode (Laporan & Analytics). Sumber: ledger
// transaksi_komisi, tanggal = saat booking pertama kali lunas.
export default function PeringkatAgenPage() {
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const [defaultDari, defaultSampai] = presetRange('bulan_ini');
  const preset = params.get('preset') || (params.get('dari') ? 'kustom' : 'bulan_ini');
  const dari = params.get('dari') || defaultDari;
  const sampai = params.get('sampai') || defaultSampai;
  const brandId = params.get('brand_id') || '';

  const [brands, setBrands] = useState([]);
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    listBrands().then((b) => setBrands(b || [])).catch(() => {});
  }, []);

  useEffect(() => {
    if (!dari || !sampai || dari > sampai) {
      setItems([]);
      setLoading(false);
      setError('Tanggal "dari" tidak boleh setelah tanggal "sampai".');
      return;
    }
    setLoading(true);
    setError('');
    getPeringkatAgen({ dari, sampai, brandId })
      .then(setItems)
      .catch((err) => setError(err.response?.data?.error || 'Peringkat agen gagal dimuat. Silakan coba kembali.'))
      .finally(() => setLoading(false));
  }, [dari, sampai, brandId]);

  const update = (changes) => {
    const next = new URLSearchParams(params);
    Object.entries(changes).forEach(([k, v]) => (v ? next.set(k, v) : next.delete(k)));
    setParams(next, { replace: true });
  };

  const pickPreset = (val) => {
    const key = val?.target ? val.target.value : val;
    if (key === 'kustom') {
      update({ preset: 'kustom', dari, sampai });
      return;
    }
    const [d, s] = presetRange(key);
    update({ preset: key, dari: d, sampai: s });
  };

  const brandsMap = useMemo(() => Object.fromEntries(brands.map((b) => [b.id, b])), [brands]);
  const ringkasan = useMemo(
    () => ({
      agen: items.length,
      pax: items.reduce((s, it) => s + it.pax_closing, 0),
      komisi: items.reduce((s, it) => s + Number(it.total_komisi || 0), 0),
    }),
    [items]
  );

  const columns = [
    { header: '#', key: 'peringkat', sortable: true },
    { header: 'Agen', key: 'nama_lengkap', sortable: true },
    { header: 'Brand', key: 'brand_name', sortable: true },
    { header: 'Pax Closing', key: 'pax_closing', sortable: true },
    { header: 'Booking', key: 'booking_closing', sortable: true },
    { header: 'Komisi Langsung', key: 'komisi_langsung', sortable: true },
    { header: 'Bonus Pembinaan', key: 'bonus_pembinaan', sortable: true },
    { header: 'Repeat Order', key: 'repeat_order', sortable: true },
    { header: 'Total Komisi', key: 'total_komisi', sortable: true },
  ];

  const renderCell = (row, key) => {
    if (key === 'peringkat') {
      return row.peringkat <= TOP
        ? <span className="text-sm font-heading font-bold text-primary-600">#{row.peringkat}</span>
        : <span className="text-[13px] font-body text-neutral-500">#{row.peringkat}</span>;
    }
    if (key === 'nama_lengkap') {
      return (
        <div className="min-w-0">
          <p className="font-heading font-semibold text-neutral-900 truncate leading-snug">{row.nama_lengkap}</p>
          <p className="text-[11px] font-body text-neutral-400">
            Kode <span className="font-mono">{row.kode_referral || '-'}</span>{row.status_agen === 'nonaktif' ? ' · nonaktif' : ''}
          </p>
        </div>
      );
    }
    if (key === 'brand_name') return <BrandChip name={brandsMap[row.brand_id]?.name || row.brand_name} color={brandsMap[row.brand_id]?.primary_color} />;
    if (['komisi_langsung', 'bonus_pembinaan', 'repeat_order'].includes(key)) return <span className="text-[13px] font-body text-neutral-700 whitespace-nowrap">{money(row[key])}</span>;
    if (key === 'total_komisi') return <span className="text-[13px] font-body font-semibold text-neutral-900 whitespace-nowrap">{money(row.total_komisi)}</span>;
    if (key === 'pax_closing' || key === 'booking_closing') return <span className="text-[13px] font-body text-neutral-700">{row[key] ?? 0}</span>;
    return row[key] ?? '-';
  };

  const toolbarActions = (
    <div className="flex flex-wrap items-center gap-2">
      <CustomDropdown
        name="preset"
        value={preset}
        onChange={pickPreset}
        placeholder="Pilih periode"
        options={[
          ...Object.entries(PRESETS).map(([value, p]) => ({ value, label: p.label })),
          { value: 'kustom', label: 'Kustom' },
        ]}
        className="!mb-0 w-full sm:w-44"
      />
      <CustomDropdown
        name="brand_id"
        value={brandId}
        onChange={(val) => update({ brand_id: val?.target ? val.target.value : val })}
        placeholder="Semua brand"
        options={[{ value: '', label: 'Semua brand' }, ...brands.map((b) => ({ value: String(b.id), label: b.name }))]}
        className="!mb-0 w-full sm:w-44"
      />
    </div>
  );
  const toolbarPanel = preset === 'kustom' && (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <Input label="Dari tanggal" type="date" name="dari" value={dari}
        onChange={(e) => update({ preset: 'kustom', dari: e.target.value, sampai })} className="!mb-0" />
      <Input label="Sampai tanggal" type="date" name="sampai" value={sampai} min={dari || undefined}
        onChange={(e) => update({ preset: 'kustom', dari, sampai: e.target.value })} className="!mb-0" />
    </div>
  );
  return (
    <div className="space-y-6">
      <PageHeader title="Peringkat Agen" />

      {error && <Alert variant="error" message={error} />}

      {!loading && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatTile icon={Users} variant="primary" value={ringkasan.agen} label="Agen berkomisi" />
          <StatTile icon={Trophy} variant="success" value={ringkasan.pax} label="Pax closing" />
          <StatTile icon={Coins} variant="violet" value={money(ringkasan.komisi)} label="Total komisi" hint="Tanpa cashback" />
        </div>
      )}

      <DataTable
        columns={columns}
        data={items}
        renderCell={renderCell}
        onRowClick={(row) => navigate(`/agen/${row.jamaah_id}`)}
        itemsPerPage={20}
        toolbarActions={toolbarActions}
        toolbarPanel={toolbarPanel}
        searchPlaceholder="Cari nama agen atau kode referral..."
        emptyMessage={loading ? 'Memuat peringkat...' : 'Belum ada komisi tercatat di periode ini'}
      />
    </div>
  );
}
