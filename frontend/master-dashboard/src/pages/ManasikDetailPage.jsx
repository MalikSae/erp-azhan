import React, { useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom';
import { ExternalLink, Trash2 } from 'lucide-react';
import PageHeader from 'shared/ui/PageHeader';
import Button from 'shared/ui/Button';
import Badge from 'shared/ui/Badge';
import Alert from 'shared/ui/Alert';
import Modal from 'shared/ui/Modal';
import Table from 'shared/ui/Table';
import Input from 'shared/ui/Input';
import Checkbox from 'shared/ui/Checkbox';
import CustomDropdown from 'shared/ui/CustomDropdown';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import EmptyState from 'shared/ui/EmptyState';
import { Surface, Section, Field, SummaryItem, TableCard } from 'shared/ui/DetailLayout';
import { AuthContext } from '../context/AuthContext';
import PilihPaketManasik, { tanggalPendek } from '../components/PilihPaketManasik';
import {
  catatHadirManasik, deleteManasik, getManasik, lepasPaketManasik, listPaketTersediaManasik, tambahPaketManasik,
} from '../api/manasik';

const jam = (hhmm) => (hhmm || '').replace(':', '.');
const valueOf = (v) => (v?.target ? v.target.value : v);
const tanggalPanjang = (iso) =>
  new Date(`${iso}T00:00:00`).toLocaleDateString('id-ID', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' });
const waktuCatat = (s) => (s ? `${tanggalPendek(s)}, ${s.slice(11, 16).replace(':', '.')}` : '');

const FILTER_KONFIRMASI = [
  { value: '', label: 'Semua konfirmasi' },
  { value: 'hadir', label: 'Konfirmasi hadir' },
  { value: 'tidak', label: 'Tidak hadir' },
  { value: 'belum', label: 'Belum konfirmasi' },
];

// Detail manasik: info, paket terkait, rekap konfirmasi hadir di app dan kehadiran nyata, serta daftar jamaah dewasa. Mulai hari
// manasik, petugas mencentang jamaah yang datang; centang itu mengisi penanda "sudah manasik" di booking jamaah.
const ManasikDetailPage = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  const { hasPermission } = useContext(AuthContext);
  const bisaUbah = hasPermission('manasik.edit');
  const [data, setData] = useState(null);
  const [loadError, setLoadError] = useState(null);
  const [errorMessage, setErrorMessage] = useState(null);
  const [successMessage, setSuccessMessage] = useState(null);
  const [konfirmasiHapus, setKonfirmasiHapus] = useState(false);
  const [lepas, setLepas] = useState(null);
  const [tambah, setTambah] = useState(null); // { paket, dipilih } saat modal terbuka
  const [modalError, setModalError] = useState(null);
  const [memproses, setMemproses] = useState(false);
  const [mencatat, setMencatat] = useState(null);
  const [filterPaket, setFilterPaket] = useState('');
  const [filterJawaban, setFilterJawaban] = useState('');
  const [cari, setCari] = useState('');

  useEffect(() => {
    if (location.state?.successMessage) {
      setSuccessMessage(location.state.successMessage);
      window.history.replaceState({}, '');
    }
  }, [location.state]);

  const load = useCallback(async () => {
    try {
      setData(await getManasik(id));
      setLoadError(null);
    } catch (err) {
      setLoadError(err.response?.data?.error || 'Gagal memuat manasik.');
    }
  }, [id]);

  useEffect(() => { load(); }, [load]);

  const jamaahTampil = useMemo(() => {
    if (!data) return [];
    const q = cari.trim().toLowerCase();
    return data.jamaah.filter((j) => {
      if (filterPaket && String(j.schedule_id) !== filterPaket) return false;
      if (filterJawaban === 'hadir' && j.jawaban !== true) return false;
      if (filterJawaban === 'tidak' && j.jawaban !== false) return false;
      if (filterJawaban === 'belum' && j.jawaban !== null) return false;
      return !q || j.nama.toLowerCase().includes(q) || j.booking_code.toLowerCase().includes(q);
    });
  }, [data, filterPaket, filterJawaban, cari]);

  if (loadError) {
    return (
      <div className="space-y-6">
        <PageHeader title="Manasik" onBack={() => navigate('/manasik')} />
        <Alert variant="error">{loadError}</Alert>
      </div>
    );
  }
  if (!data) {
    return (
      <div className="space-y-6">
        <PageHeader title="Manasik" onBack={() => navigate('/manasik')} />
        <LoadingSpinner variant="card" />
      </div>
    );
  }

  const m = data.manasik;
  const namaPaket = Object.fromEntries(data.paket.map((p) => [p.schedule_id, p.nama]));
  const hariIni = new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Jakarta' });
  const status = m.tanggal > hariIni
    ? <Badge variant="info">Akan datang</Badge>
    : m.tanggal === hariIni ? <Badge variant="warning">Hari ini</Badge> : <Badge variant="neutral">Selesai</Badge>;

  const jalankan = async (fn, pesanSukses, tutup) => {
    setMemproses(true);
    setModalError(null);
    try {
      const hasil = await fn();
      if (hasil) setData(hasil);
      tutup?.();
      if (pesanSukses) setSuccessMessage(pesanSukses);
    } catch (err) {
      setModalError(err.response?.data?.error || 'Gagal memproses manasik.');
    } finally {
      setMemproses(false);
    }
  };

  const bukaTambah = async () => {
    setModalError(null);
    setTambah({ paket: null, dipilih: [] });
    try {
      const paket = await listPaketTersediaManasik({ brand_id: m.brand_id });
      setTambah({ paket: paket || [], dipilih: [] });
    } catch (err) {
      setModalError(err.response?.data?.error || 'Gagal memuat paket.');
      setTambah({ paket: [], dipilih: [] });
    }
  };

  const catatHadir = async (j) => {
    setMencatat(j.pax_id);
    setErrorMessage(null);
    try {
      setData(await catatHadirManasik(id, j.pax_id, !j.hadir));
    } catch (err) {
      setErrorMessage(err.response?.data?.error || 'Gagal mencatat kehadiran.');
    } finally {
      setMencatat(null);
    }
  };

  const jawabanLepas = lepas ? data.jamaah.filter((j) => j.schedule_id === lepas.schedule_id && j.jawaban !== null).length : 0;
  const peta = m.gmaps_url || `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(m.alamat)}`;

  const kolomPaket = [
    { header: 'Paket', key: 'nama' },
    { header: 'Berangkat', key: 'berangkat_tanggal' },
    { header: 'Jamaah', key: 'jumlah_jamaah', align: 'right' },
    ...(bisaUbah && !data.sudah_dimulai && data.paket.length > 1 ? [{ header: '', key: 'aksi', align: 'right' }] : []),
  ];
  const selPaket = (p, key) => {
    if (key === 'nama') {
      return <Link to={`/schedules/${p.schedule_id}`} className="font-medium text-neutral-900 hover:text-primary-600">{p.nama}</Link>;
    }
    if (key === 'berangkat_tanggal') return <span className="whitespace-nowrap">{tanggalPendek(p.berangkat_tanggal)}</span>;
    if (key === 'jumlah_jamaah') return <span className="tabular-nums">{p.jumlah_jamaah}</span>;
    if (key === 'aksi') {
      return <Button size="sm" variant="danger-outline" onClick={() => { setModalError(null); setLepas(p); }}>Lepas</Button>;
    }
    return p[key];
  };

  const kolomJamaah = [
    { header: 'Jamaah', key: 'nama' },
    { header: 'Paket', key: 'paket' },
    { header: 'Konfirmasi hadir', key: 'jawaban' },
    { header: 'Kehadiran', key: 'hadir' },
  ];
  const selJamaah = (j, key) => {
    if (key === 'nama') {
      return (
        <div className="min-w-0">
          <div className="font-medium text-neutral-900">{j.nama}</div>
          <Link to={`/bookings/${j.booking_id}`} className="font-mono text-xs text-neutral-500 underline decoration-neutral-300 underline-offset-2 hover:text-neutral-900">
            {j.booking_code}
          </Link>
        </div>
      );
    }
    if (key === 'paket') return <span className="text-neutral-700">{namaPaket[j.schedule_id] || '-'}</span>;
    if (key === 'jawaban') {
      if (j.jawaban === true) return <Badge variant="success">Hadir</Badge>;
      if (j.jawaban === false) return <Badge variant="neutral">Tidak hadir</Badge>;
      return <span className="text-neutral-400" aria-label="Belum konfirmasi">–</span>;
    }
    if (key === 'hadir') {
      const catatan = j.dicatat_at ? `${j.dicatat_oleh ? `${j.dicatat_oleh}, ` : ''}${waktuCatat(j.dicatat_at)}` : '';
      if (data.absensi_terbuka && bisaUbah) {
        return (
          <label className="flex cursor-pointer items-start gap-2">
            <span className="flex h-5 items-center">
              <Checkbox
                checked={j.hadir}
                disabled={mencatat === j.pax_id}
                onChange={() => catatHadir(j)}
                ariaLabel={`Hadir ${j.nama}`}
              />
            </span>
            <span className="min-w-0">
              <span className="block text-sm text-neutral-900">{j.hadir ? 'Hadir' : 'Belum hadir'}</span>
              {catatan && <span className="block text-xs text-neutral-500">{catatan}</span>}
            </span>
          </label>
        );
      }
      if (!data.absensi_terbuka) return <span className="text-neutral-400">–</span>;
      return (
        <div>
          {j.hadir ? <Badge variant="success">Hadir</Badge> : <span className="text-neutral-400">–</span>}
          {catatan && <div className="mt-0.5 text-xs text-neutral-500">{catatan}</div>}
        </div>
      );
    }
    return j[key];
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title={m.judul}
        onBack={() => navigate('/manasik')}
        badges={status}
        meta={[`${tanggalPanjang(m.tanggal)}, ${jam(m.jam_mulai)}–${jam(m.jam_selesai)} WIB`, m.tempat, data.brand_nama]}
      >
        {bisaUbah && (
          <div className="flex items-center gap-2">
            <Button variant="secondary" onClick={() => navigate(`/manasik/${id}/ubah`)}>Ubah</Button>
            {!data.ada_absensi && (
              <Button
                variant="danger-outline"
                className="w-9 !px-0"
                icon={<Trash2 size={16} />}
                aria-label="Hapus manasik"
                title="Hapus manasik"
                onClick={() => { setModalError(null); setKonfirmasiHapus(true); }}
              />
            )}
          </div>
        )}
      </PageHeader>

      {successMessage && <Alert variant="success" onClose={() => setSuccessMessage(null)}>{successMessage}</Alert>}
      {errorMessage && <Alert variant="error" onClose={() => setErrorMessage(null)}>{errorMessage}</Alert>}

      <Surface className="grid grid-cols-2 divide-neutral-200 md:grid-cols-5 md:divide-x">
        <SummaryItem label="Jamaah" value={data.rekap.jamaah} />
        <SummaryItem label="Konfirmasi hadir" value={data.rekap.berniat_hadir} />
        <SummaryItem label="Tidak hadir" value={data.rekap.tidak_hadir} />
        <SummaryItem label="Belum konfirmasi" value={data.rekap.belum} />
        <SummaryItem label="Hadir" value={data.absensi_terbuka ? data.rekap.hadir : '-'} />
      </Surface>

      <Surface className="divide-y divide-neutral-200">
        <Section title="Informasi">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <Field label="Ustadz pengisi">{m.ustadz || '-'}</Field>
            <Field label="Tempat">
              <div>{m.tempat}</div>
              <div className="mt-0.5 text-xs font-normal text-neutral-500">{m.alamat}</div>
              <a href={peta} target="_blank" rel="noreferrer" className="mt-1 inline-flex items-center gap-1 text-xs font-medium text-neutral-900 underline decoration-neutral-300 underline-offset-2 hover:decoration-neutral-900">
                Buka peta <ExternalLink size={14} />
              </a>
            </Field>
          </div>
        </Section>
        <Section title="Agenda">
          {m.agenda.length > 0 ? (
            <ul className="space-y-1.5 font-body text-sm text-neutral-800">
              {m.agenda.map((a, i) => <li key={i}>{a}</li>)}
            </ul>
          ) : (
            <EmptyState variant="inline" message="Belum ada agenda." />
          )}
        </Section>
      </Surface>

      <TableCard
        title={`Paket (${data.paket.length})`}
        aside={bisaUbah && !data.sudah_dimulai && <Button size="sm" variant="secondary" onClick={bukaTambah}>+ Tambah Paket</Button>}
      >
        <div className="overflow-x-auto">
          <Table columns={kolomPaket} data={data.paket} renderCell={selPaket} emptyMessage="Belum ada paket." />
        </div>
      </TableCard>

      <TableCard title={`Jamaah (${data.jamaah.length})`}>
        <div className="flex flex-col gap-2 border-b border-neutral-200 p-4 md:flex-row md:items-center">
          <div className="md:w-64">
            <Input name="cari_jamaah" className="!mb-0" value={cari} onChange={(e) => setCari(e.target.value)} placeholder="Cari jamaah atau kode booking" />
          </div>
          <CustomDropdown
            value={filterPaket}
            onChange={(v) => setFilterPaket(valueOf(v))}
            className="!mb-0 md:w-56"
            options={[{ value: '', label: 'Semua paket' }, ...data.paket.map((p) => ({ value: String(p.schedule_id), label: p.nama }))]}
          />
          <CustomDropdown value={filterJawaban} onChange={(v) => setFilterJawaban(valueOf(v))} className="!mb-0 md:w-48" options={FILTER_KONFIRMASI} />
        </div>
        <div className="overflow-x-auto">
          <Table
            columns={kolomJamaah}
            data={jamaahTampil}
            renderCell={selJamaah}
            emptyMessage={data.jamaah.length === 0 ? 'Belum ada jamaah.' : 'Tidak ditemukan.'}
          />
        </div>
      </TableCard>

      <Modal
        isOpen={konfirmasiHapus}
        onClose={() => !memproses && setKonfirmasiHapus(false)}
        title="Hapus Manasik"
        size="sm"
        footer={(
          <>
            <Button variant="secondary" onClick={() => setKonfirmasiHapus(false)} disabled={memproses}>Batal</Button>
            <Button
              variant="danger"
              isLoading={memproses}
              onClick={() => jalankan(async () => { await deleteManasik(id); navigate('/manasik', { state: { successMessage: 'Manasik dihapus.' } }); })}
            >
              Hapus
            </Button>
          </>
        )}
      >
        {modalError && <div className="mb-3"><Alert variant="error">{modalError}</Alert></div>}
        <p className="font-body text-sm text-neutral-700">
          Hapus {m.judul}? Konfirmasi hadir jamaah ikut terhapus.
        </p>
      </Modal>

      <Modal
        isOpen={lepas !== null}
        onClose={() => !memproses && setLepas(null)}
        title="Lepas Paket"
        size="sm"
        footer={(
          <>
            <Button variant="secondary" onClick={() => setLepas(null)} disabled={memproses}>Batal</Button>
            <Button
              variant="danger"
              isLoading={memproses}
              onClick={() => jalankan(() => lepasPaketManasik(id, lepas.schedule_id), 'Paket dilepas.', () => setLepas(null))}
            >
              Lepas
            </Button>
          </>
        )}
      >
        {modalError && <div className="mb-3"><Alert variant="error">{modalError}</Alert></div>}
        <p className="font-body text-sm text-neutral-700">
          Lepas {lepas?.nama}?{jawabanLepas > 0 ? ` ${jawabanLepas} konfirmasi hadir ikut terhapus.` : ''}
        </p>
      </Modal>

      <Modal
        isOpen={tambah !== null}
        onClose={() => !memproses && setTambah(null)}
        title="Tambah Paket"
        footer={(
          <>
            <Button variant="secondary" onClick={() => setTambah(null)} disabled={memproses}>Batal</Button>
            <Button
              isLoading={memproses}
              disabled={!tambah?.dipilih.length}
              onClick={() => jalankan(() => tambahPaketManasik(id, tambah.dipilih), 'Paket ditambahkan.', () => setTambah(null))}
            >
              Tambah
            </Button>
          </>
        )}
      >
        {modalError && <div className="mb-3"><Alert variant="error">{modalError}</Alert></div>}
        {tambah?.paket === null ? (
          <LoadingSpinner variant="inline" />
        ) : (
          <PilihPaketManasik
            paket={tambah?.paket || []}
            value={tambah?.dipilih || []}
            onChange={(dipilih) => setTambah((t) => ({ ...t, dipilih }))}
            tanggalManasik={m.tanggal}
          />
        )}
      </Modal>
    </div>
  );
};

export default ManasikDetailPage;
