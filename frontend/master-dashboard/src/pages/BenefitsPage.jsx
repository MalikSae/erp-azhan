import { useContext, useEffect, useRef, useState } from 'react';
import { AuthContext } from '../context/AuthContext';
import { listBenefits, listBenefitIcons, createBenefit, updateBenefit, deleteBenefit, listBrandOptions } from 'shared';
import Alert from 'shared/ui/Alert';
import ActionMenu from 'shared/ui/ActionMenu';
import Badge from 'shared/ui/Badge';
import BenefitIcon from 'shared/ui/BenefitIcon';
import Button from 'shared/ui/Button';
import DataTable from 'shared/ui/DataTable';
import Modal from 'shared/ui/Modal';
import PageHeader from 'shared/ui/PageHeader';
import Select from 'shared/ui/Select';
import { Pencil, Power, Trash2 } from 'lucide-react';
import BenefitFormModal from '../components/BenefitFormModal';

// Master Benefit Paket: daftar highlight benefit (ikon + teks) per brand. Tiap paket memilih tepat empat dari
// daftar ini, dan hasilnya tampil di app dan web publik. Mengubah teks atau ikon di sini berlaku di semua paket
// yang memakainya. Benefit yang masih dipakai paket tidak bisa dihapus, hanya dinonaktifkan.
export default function BenefitsPage() {
  const { isHolding, hasPermission } = useContext(AuthContext);
  const bolehUbah = hasPermission('masterdata.edit');
  const [items, setItems] = useState([]);
  const [ikon, setIkon] = useState([]);
  const [brands, setBrands] = useState([]);
  const [brandId, setBrandId] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [form, setForm] = useState(null); // null = tertutup; {} = tambah; { id, teks, ikon } = ubah
  const [hapus, setHapus] = useState(null);
  const [saving, setSaving] = useState(false);

  // Admin holding memilih brand dulu; admin satu brand langsung melihat daftar brand-nya.
  const siapMuat = !isHolding || Boolean(brandId);

  // Nomor permintaan terakhir: jawaban permintaan lama (brand sebelumnya) tidak boleh menimpa daftar brand kini.
  const nomorMuat = useRef(0);

  const load = async () => {
    const nomor = ++nomorMuat.current;
    if (!siapMuat) return setItems([]);
    setLoading(true);
    setError('');
    try {
      const data = await listBenefits(isHolding ? Number(brandId) : undefined);
      if (nomor === nomorMuat.current) setItems(data);
    } catch (err) {
      if (nomor === nomorMuat.current) setError(err.response?.data?.error || 'Benefit gagal dimuat. Silakan coba kembali.');
    } finally {
      if (nomor === nomorMuat.current) setLoading(false);
    }
  };

  useEffect(() => {
    listBenefitIcons().then(setIkon).catch(() => {});
    if (isHolding) {
      listBrandOptions()
        .then((d) => {
          const daftar = Array.isArray(d) ? d : [];
          setBrands(daftar);
          if (daftar.length > 0) setBrandId((lama) => lama || String(daftar[0].id));
        })
        .catch(() => {});
    }
  }, [isHolding]);

  useEffect(() => {
    // Baris brand sebelumnya dibuang dulu, supaya aksi tidak mengenai benefit brand lain selama memuat.
    setItems([]);
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [brandId, isHolding]);

  const simpan = async ({ teks, ikon: kunci }) => {
    if (form.id) await updateBenefit(form.id, { teks, ikon: kunci });
    else await createBenefit({ brand_id: isHolding ? Number(brandId) : undefined, teks, ikon: kunci });
    setForm(null);
    await load();
  };

  const ubahAktif = async (row) => {
    setError('');
    try {
      await updateBenefit(row.id, { teks: row.teks, ikon: row.ikon, is_active: !row.is_active });
      await load();
    } catch (err) {
      setError(err.response?.data?.error || 'Status benefit gagal diubah.');
    }
  };

  const konfirmasiHapus = async () => {
    setSaving(true);
    try {
      await deleteBenefit(hapus.id);
      setHapus(null);
      await load();
    } catch (err) {
      setError(err.response?.data?.error || 'Benefit gagal dihapus.');
      setHapus(null);
    } finally {
      setSaving(false);
    }
  };

  const columns = [
    { header: 'Benefit', key: 'teks', sortable: true },
    { header: 'Dipakai', key: 'dipakai', sortable: true },
    { header: 'Status', key: 'is_active' },
    ...(bolehUbah ? [{ header: 'Aksi', key: 'actions' }] : []),
  ];

  const renderCell = (row, key) => {
    if (key === 'teks') {
      return (
        <span className="flex items-center gap-3">
          <BenefitIcon name={row.ikon} className="text-neutral-700" />
          <span className="font-body font-medium text-neutral-900">{row.teks}</span>
        </span>
      );
    }
    if (key === 'dipakai') return <span className="tabular-nums">{row.dipakai > 0 ? `${row.dipakai} paket` : <span className="text-neutral-500">Belum dipakai</span>}</span>;
    if (key === 'is_active') return <Badge variant={row.is_active ? 'success' : 'neutral'}>{row.is_active ? 'Aktif' : 'Nonaktif'}</Badge>;
    if (key === 'actions') {
      return <ActionMenu align="right" items={[
        { label: 'Ubah', icon: Pencil, onClick: () => setForm({ id: row.id, teks: row.teks, ikon: row.ikon }) },
        { label: row.is_active ? 'Nonaktifkan' : 'Aktifkan', icon: Power, onClick: () => ubahAktif(row) },
        {
          label: 'Hapus', icon: Trash2, danger: true, onClick: () => setHapus(row),
          disabled: row.dipakai > 0, disabledTooltip: 'Masih dipakai paket. Nonaktifkan saja.',
        },
      ]} />;
    }
    return row[key] ?? '-';
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Benefit Paket"
        actionLabel={bolehUbah && siapMuat ? 'Tambah benefit' : undefined}
        onAction={() => setForm({})}
      />
      {error && <Alert variant="error" message={error} />}

      <DataTable
        columns={columns}
        data={items}
        renderCell={renderCell}
        itemsPerPage={20}
        searchPlaceholder="Cari teks benefit..."
        loading={loading}
        emptyMessage={siapMuat ? 'Belum ada benefit. Tambahkan benefit untuk dipilih di form paket.' : 'Pilih brand untuk melihat daftar benefit.'}
        toolbarLeading={
          isHolding ? (
            <div role="group" aria-label="Brand">
              <Select className="!mb-0 w-56" value={brandId} onChange={setBrandId} placeholder="Pilih brand" options={brands.map((b) => ({ value: String(b.id), label: b.name || b.nama }))} />
            </div>
          ) : undefined
        }
      />

      <BenefitFormModal
        isOpen={Boolean(form)}
        onClose={() => setForm(null)}
        awal={form?.id ? form : null}
        ikon={ikon}
        onSimpan={simpan}
      />

      <Modal
        isOpen={Boolean(hapus)}
        onClose={() => !saving && setHapus(null)}
        title="Hapus Benefit"
        size="sm"
        footer={
          <div className="flex flex-col-reverse sm:flex-row sm:justify-end gap-2 w-full">
            <Button variant="secondary" onClick={() => setHapus(null)} disabled={saving}>Batal</Button>
            <Button variant="danger" onClick={konfirmasiHapus} isLoading={saving}>Hapus</Button>
          </div>
        }
      >
        <p className="text-sm text-neutral-700">
          Hapus benefit “{hapus?.teks}”? Benefit yang masih dipakai paket tidak bisa dihapus; nonaktifkan saja supaya tidak muncul lagi di pilihan paket baru.
        </p>
      </Modal>
    </div>
  );
}
