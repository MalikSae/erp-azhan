import { useContext, useEffect, useState } from 'react';
import { AuthContext } from '../context/AuthContext';
import { listPengumuman, createPengumuman, updatePengumuman, deletePengumuman, listBrandOptions, listSchedulesAdmin } from 'shared';
import Alert from 'shared/ui/Alert';
import ActionMenu from 'shared/ui/ActionMenu';
import Badge from 'shared/ui/Badge';
import Button from 'shared/ui/Button';
import DataTable from 'shared/ui/DataTable';
import Input from 'shared/ui/Input';
import Modal from 'shared/ui/Modal';
import PageHeader from 'shared/ui/PageHeader';
import Select from 'shared/ui/Select';
import Textarea from 'shared/ui/Textarea';
import { Pencil, Trash2 } from 'lucide-react';

const tgl = (v) => (!v ? '-' : new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric' }).format(new Date(`${v}T00:00:00`)));

// Nada Badge per status pengumuman (satu gaya status, lihat Badge.jsx).
const STATUS = {
  tayang: { label: 'Tayang', variant: 'success' },
  akan_tayang: { label: 'Akan tayang', variant: 'warning' },
  berakhir: { label: 'Berakhir', variant: 'neutral' },
};

const StatusDot = ({ status }) => {
  const meta = STATUS[status] || { label: status, variant: 'neutral' };
  return <Badge variant={meta.variant}>{meta.label}</Badge>;
};

const KOSONG = { brand_id: '', schedule_id: '', judul: '', isi: '', mulai: '', akhir: '' };

const bulanDari = (s) => String(s.berangkat_tanggal || '').slice(0, 7); // YYYY-MM
const labelBulan = (ym) => new Intl.DateTimeFormat('id-ID', { month: 'long', year: 'numeric' }).format(new Date(`${ym}-01T00:00:00`));

// Pengumuman: teks singkat untuk pelanggan di dashboard akun. Tanpa jadwal berarti untuk semua pelanggan brand; dengan jadwal hanya untuk
// akun yang punya booking di keberangkatan itu. Tampil selama masa tayang.
export default function PengumumanPage() {
  const { isHolding } = useContext(AuthContext);
  const [items, setItems] = useState([]);
  const [brands, setBrands] = useState([]);
  const [jadwal, setJadwal] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [form, setForm] = useState(null); // null = tertutup; { id?, ...isian }
  const [bulan, setBulan] = useState(''); // saringan bulan berangkat untuk memilih jadwal (YYYY-MM), tidak ikut disimpan
  const [hapus, setHapus] = useState(null);
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState('');

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      setItems(await listPengumuman());
    } catch {
      setError('Pengumuman gagal dimuat. Silakan coba kembali.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    listSchedulesAdmin().then((d) => setJadwal(Array.isArray(d) ? d : d?.items || [])).catch(() => {});
    if (isHolding) listBrandOptions().then((d) => setBrands(Array.isArray(d) ? d : [])).catch(() => {});
  }, [isHolding]);

  const set = (k) => (e) => setForm((f) => ({ ...f, [k]: e.target.value }));
  // Admin holding melihat jadwal semua brand dari API, jadi daftar dibatasi ke brand yang dipilih (kosong sebelum brand dipilih). Admin satu
  // brand sudah menerima jadwal brand-nya saja; server juga menolak jadwal dari brand lain.
  const perluPilihBrand = isHolding && !form?.brand_id;
  const jadwalBrand = jadwal.filter((s) => !isHolding || (form?.brand_id && String(s.brand_id) === String(form.brand_id)));
  // Saringan bulan berangkat memudahkan memilih jadwal: hanya bulan yang punya jadwal di brand ini, urut dari yang terdekat.
  const opsiBulan = [...new Set(jadwalBrand.map(bulanDari).filter(Boolean))].sort().map((ym) => ({ value: ym, label: labelBulan(ym) }));
  const opsiJadwal = jadwalBrand
    .filter((s) => !bulan || bulanDari(s) === bulan)
    .sort((a, b) => String(a.berangkat_tanggal).localeCompare(String(b.berangkat_tanggal)))
    .map((s) => ({ value: String(s.id), label: `${s.jadwal_nama}${s.berangkat_tanggal ? ` (${tgl(String(s.berangkat_tanggal).slice(0, 10))})` : ''}` }));

  const bukaBaru = () => { setFormError(''); setBulan(''); setForm({ ...KOSONG }); };
  const bukaUbah = (p) => {
    setFormError('');
    const terpilih = jadwal.find((s) => s.id === p.schedule_id);
    setBulan(terpilih ? bulanDari(terpilih) : '');
    setForm({ id: p.id, brand_id: String(p.brand_id), schedule_id: p.schedule_id ? String(p.schedule_id) : '', judul: p.judul, isi: p.isi, mulai: p.mulai, akhir: p.akhir });
  };

  const simpan = async () => {
    if (isHolding && !form.id && !form.brand_id) return setFormError('Pilih brand.');
    setSaving(true);
    setFormError('');
    const payload = {
      brand_id: form.brand_id ? Number(form.brand_id) : undefined,
      schedule_id: form.schedule_id ? Number(form.schedule_id) : null,
      judul: form.judul, isi: form.isi, mulai: form.mulai, akhir: form.akhir,
    };
    try {
      if (form.id) await updatePengumuman(form.id, payload);
      else await createPengumuman(payload);
      setForm(null);
      await load();
    } catch (err) {
      setFormError(err.response?.data?.error || 'Pengumuman gagal disimpan.');
    } finally {
      setSaving(false);
    }
  };

  const konfirmasiHapus = async () => {
    setSaving(true);
    try {
      await deletePengumuman(hapus.id);
      setHapus(null);
      await load();
    } catch (err) {
      setError(err.response?.data?.error || 'Pengumuman gagal dihapus.');
      setHapus(null);
    } finally {
      setSaving(false);
    }
  };

  const columns = [
    { header: 'Judul', key: 'judul', sortable: true },
    { header: 'Untuk', key: 'jadwal_nama' },
    { header: 'Masa tayang', key: 'mulai', sortable: true },
    { header: 'Status', key: 'status' },
    { header: 'Aksi', key: 'actions' },
  ];

  const renderCell = (row, key) => {
    if (key === 'judul') return <span className="block max-w-sm truncate font-heading font-semibold text-neutral-900" title={row.judul}>{row.judul}</span>;
    if (key === 'jadwal_nama') return row.jadwal_nama || <span className="text-neutral-500">Semua pelanggan</span>;
    if (key === 'mulai') return `${tgl(row.mulai)} - ${tgl(row.akhir)}`;
    if (key === 'status') return <StatusDot status={row.status} />;
    if (key === 'actions') {
      return <ActionMenu align="right" items={[
        { label: 'Ubah', icon: Pencil, onClick: () => bukaUbah(row) },
        { label: 'Hapus', icon: Trash2, onClick: () => setHapus(row) },
      ]} />;
    }
    return row[key] ?? '-';
  };

  return (
    <div className="space-y-6">
      <PageHeader title="Pengumuman" actionLabel="Tambah pengumuman" onAction={bukaBaru} />
      {error && <Alert variant="error" message={error} />}

      <DataTable
        columns={columns}
        data={items}
        renderCell={renderCell}
        itemsPerPage={10}
        searchPlaceholder="Cari judul pengumuman..."
        emptyMessage={loading ? 'Memuat pengumuman...' : 'Belum ada pengumuman'}
      />

      <Modal
        isOpen={Boolean(form)}
        onClose={() => !saving && setForm(null)}
        title={form?.id ? 'Ubah Pengumuman' : 'Tambah Pengumuman'}
        size="lg"
        footer={
          <div className="flex flex-col-reverse sm:flex-row sm:justify-end gap-2 w-full">
            <Button variant="secondary" onClick={() => setForm(null)} disabled={saving}>Batal</Button>
            <Button onClick={simpan} isLoading={saving}>Simpan</Button>
          </div>
        }
      >
        {form && (
          <div className="space-y-1">
            {formError && <Alert variant="error" message={formError} />}
            {isHolding && !form.id && (
              <Select label="Brand" required value={form.brand_id} onChange={(v) => { setBulan(''); setForm((f) => ({ ...f, brand_id: v, schedule_id: '' })); }}
                options={brands.map((b) => ({ value: String(b.id), label: b.name || b.nama }))} />
            )}
            <Input label="Judul" required value={form.judul} onChange={set('judul')} maxLength={120} placeholder="Mis. Jadwal manasik kelompok" />
            <Textarea label="Isi" required value={form.isi} onChange={set('isi')} maxLength={1000} rows={4} placeholder="Satu atau dua kalimat singkat" />
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-4">
              <Input label="Mulai tayang" type="date" required value={form.mulai} onChange={set('mulai')} />
              <Input label="Akhir tayang" type="date" required value={form.akhir} onChange={set('akhir')} />
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-4">
              <Select label="Bulan berangkat" value={bulan} disabled={perluPilihBrand} placeholder="Pilih brand dulu"
                options={perluPilihBrand ? [] : [{ value: '', label: 'Semua bulan' }, ...opsiBulan]}
                onChange={(ym) => {
                  setBulan(ym);
                  // Jadwal terpilih yang tidak berada di bulan itu dilepas.
                  setForm((f) => (ym && f.schedule_id && bulanDari(jadwal.find((s) => String(s.id) === f.schedule_id) || {}) !== ym ? { ...f, schedule_id: '' } : f));
                }} />
              <Select label="Jadwal keberangkatan" value={form.schedule_id} disabled={perluPilihBrand} placeholder="Pilih brand dulu"
                options={perluPilihBrand ? [] : [{ value: '', label: 'Semua pelanggan brand ini' }, ...opsiJadwal]}
                onChange={(v) => setForm((f) => ({ ...f, schedule_id: v }))} />
            </div>
            <p className="text-xs text-neutral-500">Pilih bulan berangkat untuk mempersempit daftar jadwal. Kosongkan jadwal agar pengumuman tampil untuk semua pelanggan; pilih jadwal bila hanya untuk jamaah di keberangkatan itu.</p>
          </div>
        )}
      </Modal>

      <Modal
        isOpen={Boolean(hapus)}
        onClose={() => !saving && setHapus(null)}
        title="Hapus Pengumuman"
        size="md"
        footer={
          <div className="flex flex-col-reverse sm:flex-row sm:justify-end gap-2 w-full">
            <Button variant="secondary" onClick={() => setHapus(null)} disabled={saving}>Batal</Button>
            <Button variant="danger" onClick={konfirmasiHapus} isLoading={saving}>Hapus</Button>
          </div>
        }
      >
        <p className="text-sm text-neutral-700">Hapus pengumuman “{hapus?.judul}”? Pengumuman langsung hilang dari dashboard pelanggan dan tidak dapat dikembalikan.</p>
      </Modal>
    </div>
  );
}
