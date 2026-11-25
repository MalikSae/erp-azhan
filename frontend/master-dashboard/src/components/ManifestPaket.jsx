import React, { useContext, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Download } from 'lucide-react';
import Button from 'shared/ui/Button';
import Badge from 'shared/ui/Badge';
import Input from 'shared/ui/Input';
import Textarea from 'shared/ui/Textarea';
import Alert from 'shared/ui/Alert';
import Table from 'shared/ui/Table';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import { Surface, Section, TableCard } from 'shared/ui/DetailLayout';
import { getManifest, simpanManifest, unduhManifest } from '../api/manifest';
import { AuthContext } from '../context/AuthContext';

// Tab Manifest di Detail Paket (Fase 1): PNR dan detail penerbangan, pemeriksaan data pax, unduh .xlsx.

const BULAN = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];

const tanggalPendek = (iso) => {
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(iso || '');
  return m ? `${Number(m[3])} ${BULAN[Number(m[2]) - 1]} ${m[1]}` : '-';
};

const waktuPendek = (iso) => {
  const m = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})/.exec(iso || '');
  return m ? `${Number(m[3])} ${BULAN[Number(m[2]) - 1]} ${m[4]}.${m[5]}` : '';
};

const namaTampil = (p) => (p.nama_paspor || p.nama || '').toUpperCase();

// Pesan galat dari respons blob (unduhan) atau JSON biasa.
const pesanGalat = async (err, cadangan) => {
  const data = err?.response?.data;
  if (data instanceof Blob) {
    try {
      return JSON.parse(await data.text()).error || cadangan;
    } catch {
      return cadangan;
    }
  }
  return data?.error || cadangan;
};

const KOLOM = [
  { header: 'No', key: 'no' },
  { header: 'Nama di paspor', key: 'nama' },
  { header: 'L/P', key: 'jenis_kelamin' },
  { header: 'Tanggal lahir', key: 'tanggal_lahir' },
  { header: 'Paspor', key: 'paspor' },
  { header: 'Tiket', key: 'jenis_tiket' },
  { header: 'Data', key: 'masalah' },
];

const ManifestPaket = ({ scheduleId }) => {
  const { hasPermission } = useContext(AuthContext);
  // Tautan ke Detail Jamaah hanya untuk admin yang boleh membukanya; selain itu teks biasa.
  const bolehLihatJamaah = hasPermission('jamaah.view');
  const [data, setData] = useState(null);
  const [loadError, setLoadError] = useState(null);
  const [form, setForm] = useState({ pnr: '', detail_penerbangan: '' });
  const [menyimpan, setMenyimpan] = useState(false);
  const [mengunduh, setMengunduh] = useState(false);
  const [pesan, setPesan] = useState(null);

  const terapkan = (d) => {
    setData(d);
    setForm({ pnr: d.paket.pnr || '', detail_penerbangan: d.paket.detail_penerbangan || '' });
  };

  useEffect(() => {
    let batal = false;
    setLoadError(null);
    getManifest(scheduleId)
      .then((d) => { if (!batal) terapkan(d); })
      .catch((err) => { if (!batal) setLoadError(err.response?.data?.error || 'Manifest gagal dimuat.'); });
    return () => { batal = true; };
  }, [scheduleId]);

  if (loadError) return <Alert variant="error">{loadError}</Alert>;
  if (!data) return <LoadingSpinner variant="card" />;

  const berubah = form.pnr !== (data.paket.pnr || '') || form.detail_penerbangan !== (data.paket.detail_penerbangan || '');

  const simpan = async () => {
    setMenyimpan(true);
    setPesan(null);
    try {
      terapkan(await simpanManifest(scheduleId, form));
      setPesan({ variant: 'success', teks: 'Penerbangan tersimpan.' });
    } catch (err) {
      setPesan({ variant: 'error', teks: err.response?.data?.error || 'Gagal menyimpan.' });
    } finally {
      setMenyimpan(false);
    }
  };

  const unduh = async () => {
    setMengunduh(true);
    setPesan(null);
    try {
      if (berubah) terapkan(await simpanManifest(scheduleId, form));
      await unduhManifest(scheduleId);
      setData(await getManifest(scheduleId));
    } catch (err) {
      setPesan({ variant: 'error', teks: await pesanGalat(err, 'Gagal mengunduh manifest.') });
    } finally {
      setMengunduh(false);
    }
  };

  const renderCell = (row, key, idx) => {
    switch (key) {
      case 'no':
        return <span className="tabular-nums text-neutral-500">{idx + 1}</span>;
      case 'nama':
        return (
          <div className="min-w-0">
            {bolehLihatJamaah ? (
              <Link to={`/jamaah/${row.jamaah_id}`} className="font-medium text-neutral-900 hover:text-primary-600">
                {namaTampil(row)}
              </Link>
            ) : (
              <span className="font-medium text-neutral-900">{namaTampil(row)}</span>
            )}
            <div className="flex items-center gap-2 text-xs text-neutral-500">
              <span className="font-mono">{row.kode_booking || `#${row.booking_id}`}</span>
              {row.status_booking === 'baru' && <Badge variant="warning">Belum DP</Badge>}
            </div>
          </div>
        );
      case 'jenis_kelamin':
        return row.jenis_kelamin || '-';
      case 'tanggal_lahir':
        return <span className="tabular-nums">{tanggalPendek(row.tanggal_lahir)}</span>;
      case 'paspor':
        return (
          <div>
            <div className="font-mono text-xs text-neutral-900">{row.no_paspor || '-'}</div>
            <div className="text-xs text-neutral-500">{row.paspor_habis ? `s.d. ${tanggalPendek(row.paspor_habis)}` : ''}</div>
          </div>
        );
      case 'jenis_tiket':
        return <span className="text-xs">{row.jenis_tiket}</span>;
      case 'masalah':
        return row.masalah.length === 0 ? (
          <Badge variant="success">Lengkap</Badge>
        ) : (
          <div className="flex flex-col items-start gap-1">
            {row.masalah.map((m) => (bolehLihatJamaah ? (
              <Link key={m} to={`/jamaah/${row.jamaah_id}`} className="hover:opacity-80">
                <Badge variant="warning">{m}</Badge>
              </Link>
            ) : (
              <Badge key={m} variant="warning">{m}</Badge>
            )))}
          </div>
        );
      default:
        return null;
    }
  };

  const adaPax = data.pax.length > 0;
  const jejakUnduh = data.terakhir_unduh
    ? `Terakhir diunduh ${data.terakhir_unduh.nama}, ${waktuPendek(data.terakhir_unduh.waktu)}`
    : '';

  return (
    <div className="space-y-6">
      {pesan && <Alert variant={pesan.variant} onClose={() => setPesan(null)}>{pesan.teks}</Alert>}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
        <div className="lg:col-span-2 space-y-4 min-w-0">
          {data.jumlah_masalah > 0 && (
            <Alert variant="warning">{data.jumlah_masalah} pax datanya belum lengkap.</Alert>
          )}
          <TableCard
            title={`Pax ${data.pax.length}`}
            aside={adaPax && (
              <div className="flex items-center gap-3">
                {jejakUnduh && <span className="hidden sm:inline text-xs font-body text-neutral-500">{jejakUnduh}</span>}
                <Button onClick={unduh} isLoading={mengunduh} icon={<Download size={16} strokeWidth={1.75} />}>
                  Unduh manifest
                </Button>
              </div>
            )}
          >
            <Table columns={KOLOM} data={data.pax} renderCell={renderCell} emptyMessage="Belum ada pax." />
          </TableCard>
          {jejakUnduh && <p className="sm:hidden text-xs font-body text-neutral-500">{jejakUnduh}</p>}
        </div>

        <Surface className="order-first lg:order-none">
          <Section title="Penerbangan">
            <div className="space-y-4">
              <Input
                label="PNR"
                name="pnr"
                value={form.pnr}
                onChange={(e) => setForm((f) => ({ ...f, pnr: e.target.value }))}
                placeholder="97BC5W"
                maxLength={100}
              />
              <Textarea
                label="Detail penerbangan"
                name="detail_penerbangan"
                value={form.detail_penerbangan}
                onChange={(e) => setForm((f) => ({ ...f, detail_penerbangan: e.target.value }))}
                placeholder={'SV 827 14SEP 1 CGKJED 0040 0610\nSV 826 22SEP 2 JEDCGK 0840 2245'}
                rows={4}
                maxLength={2000}
              />
              {berubah && (
                <Button variant="secondary" className="w-full justify-center" onClick={simpan} isLoading={menyimpan}>
                  Simpan
                </Button>
              )}
            </div>
          </Section>
        </Surface>
      </div>
    </div>
  );
};

export default ManifestPaket;
