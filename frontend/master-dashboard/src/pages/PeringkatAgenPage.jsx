import { useEffect, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Coins, Trophy, Users } from 'lucide-react';
import { BrandCell, getPeringkatAgen, listBrands, money } from 'shared';
import Alert from '../components/ui/Alert';
import Badge from '../components/ui/Badge';
import CustomDropdown from '../components/ui/CustomDropdown';
import DataTable from '../components/ui/DataTable';
import Input from '../components/ui/Input';
import PageHeader from '../components/ui/PageHeader';

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
        ? <Badge variant="primary">#{row.peringkat}</Badge>
        : <span className="text-neutral-600">#{row.peringkat}</span>;
    }
    if (key === 'nama_lengkap') {
      return (
        <div className="min-w-0">
          <p className="font-semibold text-neutral-900 truncate">{row.nama_lengkap}</p>
          <p className="text-xs text-neutral-500 font-mono">
            {row.kode_referral || '-'}{row.status_agen === 'nonaktif' ? ' · nonaktif' : ''}
          </p>
        </div>
      );
    }
    if (key === 'brand_name') return <BrandCell brand={brandsMap[row.brand_id]} brandName={row.brand_name} showText />;
    if (['komisi_langsung', 'bonus_pembinaan', 'repeat_order'].includes(key)) return money(row[key]);
    if (key === 'total_komisi') return <span className="font-bold text-neutral-900">{money(row.total_komisi)}</span>;
    return row[key] ?? '-';
  };

  const Stat = ({ icon: Icon, label, value }) => (
    <div className="rounded-xl border border-neutral-200 bg-white p-4 flex items-center gap-3">
      <div className="w-10 h-10 shrink-0 rounded-xl bg-primary-50 text-primary-700 flex items-center justify-center">
        <Icon size={18} />
      </div>
      <div className="min-w-0">
        <p className="text-xs text-neutral-500">{label}</p>
        <p className="text-lg font-bold text-neutral-900 truncate">{value}</p>
      </div>
    </div>
  );

  return (
    <div className="space-y-5">
      <PageHeader
        title="Peringkat Agen"
        subtitle="Kinerja agen Syiar per periode, diurutkan dari total komisi. Tanggal mengacu saat booking lunas dan komisi tercatat; cashback tidak dihitung."
      />

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
        <CustomDropdown
          label="Periode"
          name="preset"
          value={preset}
          onChange={pickPreset}
          options={[
            ...Object.entries(PRESETS).map(([value, p]) => ({ value, label: p.label })),
            { value: 'kustom', label: 'Kustom' },
          ]}
        />
        <Input label="Dari tanggal" type="date" name="dari" value={dari} onChange={(e) => update({ preset: 'kustom', dari: e.target.value, sampai })} />
        <Input label="Sampai tanggal" type="date" name="sampai" value={sampai} onChange={(e) => update({ preset: 'kustom', dari, sampai: e.target.value })} />
        <CustomDropdown
          label="Brand"
          name="brand_id"
          value={brandId}
          onChange={(val) => update({ brand_id: val?.target ? val.target.value : val })}
          placeholder="Semua brand"
          options={[{ value: '', label: 'Semua brand' }, ...brands.map((b) => ({ value: String(b.id), label: b.name }))]}
        />
      </div>

      {error && <Alert variant="error" message={error} />}

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <Stat icon={Users} label="Agen berkomisi" value={ringkasan.agen} />
        <Stat icon={Trophy} label="Pax closing" value={ringkasan.pax} />
        <Stat icon={Coins} label="Total komisi" value={money(ringkasan.komisi)} />
      </div>

      <DataTable
        columns={columns}
        data={items}
        renderCell={renderCell}
        onRowClick={(row) => navigate(`/agen/${row.jamaah_id}`)}
        itemsPerPage={20}
        searchPlaceholder="Cari nama agen atau kode referral..."
        emptyMessage={loading ? 'Memuat peringkat...' : 'Belum ada komisi tercatat di periode ini'}
      />
    </div>
  );
}
