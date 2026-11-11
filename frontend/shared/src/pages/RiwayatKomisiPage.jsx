import { useEffect, useState } from 'react';
import { Banknote, CheckCircle2, Clock3 } from 'lucide-react';
import { useSearchParams } from 'react-router-dom';
import { listAgen, listRiwayatKomisi } from '../api/agen';
import { listBrandOptions } from '../api/brands';
import Alert from '../components/ui/Alert';
import CustomDropdown from '../components/ui/CustomDropdown';
import PeriodePicker from '../components/ui/PeriodePicker';
import DataTable from '../components/ui/DataTable';
import PageHeader from '../components/ui/PageHeader';
import StatTile from '../components/ui/StatTile';
import { dateLabel, HARI_TAHAN_KOMISI, JENIS_KOMISI, JENIS_KOMISI_FILTER, money, SEBAB_BATAL_KOMISI } from '../utils/agen';

const PER_PAGE = 20;
const valueOf = (v) => (v?.target ? v.target.value : v);

const KETERSEDIAAN_STYLE = {
  tersedia: { label: 'Dapat Dicairkan', text: 'text-success-700', dot: 'bg-success-500' },
  tertahan: { label: 'Tertahan', text: 'text-warning-700', dot: 'bg-warning-500' },
  kredit: { label: 'Kredit cashback', text: 'text-neutral-500', dot: 'bg-neutral-300' },
  batal: { label: 'Dibatalkan', text: 'text-neutral-500', dot: 'bg-neutral-400' },
};

// Screen B3: ledger transaksi_komisi lintas agen, hanya baca. Paginasi, pencarian, dan ringkasan dihitung
// server, jadi angka di kartu mencakup seluruh hasil filter, bukan hanya halaman yang tampil.
// Komisi mengikuti lunas terakhir: baris yang batal (booking turun dari lunas) tidak dihitung dan hanya
// tampil lewat filter status. showBrandColumn: pengguna holding, dengan filter brand.
// `judul` dan `subnav` (pemilih tab) diisi halaman pembungkus bertab (menu Komisi).
export default function RiwayatKomisiPage({ showBrandColumn = false, judul = 'Riwayat Komisi', subnav }) {
  const [params, setParams] = useSearchParams();
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

  const filterKey = params.toString();
  useEffect(() => {
    let batal = false;
    setLoading(true);
    setError('');
    // Tanpa saringan status: tabel memuat semua komisi, statusnya terbaca di kolom Status.
    listRiwayatKomisi({ ...filter, status: filter.status || 'semua', q, limit: PER_PAGE, offset: (page - 1) * PER_PAGE })
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
    { header: 'Tanggal Komisi', key: 'created_at' },
    { header: 'Penerima', key: 'penerima_nama' },
    { header: 'Sumber', key: 'sumber_nama' },
    { header: 'Nominal', key: 'nominal' },
    { header: 'Status', key: 'ketersediaan' },
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
            <p className="mt-0.5 text-[11px] font-body text-neutral-400">Dapat dicairkan {dateLabel(row.tersedia_mulai)}</p>
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

  // Periode = rentang tanggal komisi tercatat, dipilih di satu kontrol (PeriodePicker).
  const periode = params.get('periode') || (filter.dari && filter.sampai ? 'kustom' : '');
  const pilihPeriode = ({ preset, dari, sampai }) => ubahParams((next) => {
    ['periode', 'dari', 'sampai'].forEach((k) => next.delete(k));
    if (!preset) return;
    next.set('periode', preset);
    next.set('dari', dari);
    next.set('sampai', sampai);
  });
  // Saringan satu agen hanya datang dari tautan halaman agen ("Lihat riwayat lengkap"); di sini cukup
  // ditampilkan dan bisa dilepas. Mencari agen lain memakai kotak cari.
  const agenTersaring = filter.agen_id ? agenList.find((a) => String(a.akun_id) === filter.agen_id) : null;

  const toolbarActions = (
    <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto">
      <PeriodePicker
        value={{ preset: periode, dari: filter.dari, sampai: filter.sampai }}
        onChange={pilihPeriode}
        labelSemua="Semua periode"
        className="flex-1 sm:w-48 sm:flex-none"
      />
      <CustomDropdown
        value={filter.jenis}
        onChange={(v) => setFilter('jenis', valueOf(v))}
        options={[{ value: '', label: 'Semua jenis' }, ...Object.entries(JENIS_KOMISI_FILTER).map(([value, label]) => ({ value, label }))]}
        placeholder="Semua jenis"
        className="!mb-0 flex-1 sm:w-40 sm:flex-none"
      />
      {showBrandColumn && brands.length > 0 && (
        <CustomDropdown
          name="brand_id"
          value={filter.brand_id}
          onChange={(v) => setFilter('brand_id', valueOf(v))}
          options={[{ value: '', label: 'Semua brand' }, ...brands.map((b) => ({ value: String(b.id), label: b.name }))]}
          placeholder="Semua brand"
          className="!mb-0 flex-1 sm:w-40 sm:flex-none"
        />
      )}
    </div>
  );
  const toolbarPanel = filter.agen_id && (
    <span className="inline-flex h-10 items-center gap-2 rounded-lg border border-primary-200 bg-primary-50 px-3 text-[13px] font-body text-primary-700">
      Agen: {agenTersaring?.nama_lengkap || `#${filter.agen_id}`}
      <button type="button" onClick={() => setFilter('agen_id', '')} className="font-semibold underline underline-offset-2 cursor-pointer">Lepas</button>
    </span>
  );

  const r = hasil.ringkasan || {};
  return (
    <div className="space-y-6">
      <PageHeader title={judul} />
      {subnav}
      {error && <Alert variant="error" message={error} />}

      {/* Ringkasan baris komisi yang lolos filter. Total = Dapat Dicairkan + Tertahan. Labelnya sama dengan
          kolom Status di tabel dan kartu saldo di halaman agen. */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <StatTile icon={Banknote} variant="violet" value={money(r.total)} label="Total Komisi" hint={`${r.jumlah ?? 0} transaksi`} />
        <StatTile icon={CheckCircle2} variant="success" value={money(r.matang)} label="Dapat Dicairkan" hint={`Lewat H+${HARI_TAHAN_KOMISI} keberangkatan`} />
        <StatTile icon={Clock3} variant="warning" value={money(r.belum_matang)} label="Tertahan" hint={`Hingga H+${HARI_TAHAN_KOMISI} keberangkatan`} />
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
