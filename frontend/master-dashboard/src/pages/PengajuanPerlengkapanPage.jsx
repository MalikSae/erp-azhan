import React, { useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { AuthContext } from '../context/AuthContext';
import PageHeader from 'shared/ui/PageHeader';
import DataTable from 'shared/ui/DataTable';
import Button from 'shared/ui/Button';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import CustomDropdown from 'shared/ui/CustomDropdown';
import { listPengajuanPerlengkapan, serahkanPerlengkapan } from '../api/perlengkapan';
import { listBrandOptions } from '../api/brands';

const BULAN = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
const tanggal = (iso) => {
  if (!iso) return '-';
  const [y, m, d] = iso.slice(0, 10).split('-').map(Number);
  return `${d} ${BULAN[m - 1]} ${y}`;
};
const waktu = (s) => (s ? `${tanggal(s)}, ${s.slice(11, 16).replace(':', '.')}` : '-');
const valueOf = (v) => (v?.target ? v.target.value : v);

// Antrean pengajuan perlengkapan dari app Hana: jamaah yang perlengkapannya sudah diajukan akun tetapi belum diserahkan.
// Petugas menekan Serahkan saat barang diberikan; akun lalu mengonfirmasi penerimaan di app.
const PengajuanPerlengkapanPage = () => {
  const { isHolding, hasPermission } = useContext(AuthContext);
  const bisaSerahkan = hasPermission('booking.edit');
  const [searchParams, setSearchParams] = useSearchParams();
  const scheduleId = searchParams.get('jadwal') || '';
  const brandId = searchParams.get('brand') || '';
  const [rows, setRows] = useState([]);
  const [brands, setBrands] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  const [notice, setNotice] = useState(null);
  const [memproses, setMemproses] = useState(null);

  const setFilter = (key, value) => {
    const next = new URLSearchParams(searchParams);
    if (value) next.set(key, value); else next.delete(key);
    setSearchParams(next, { replace: true });
  };

  useEffect(() => {
    if (isHolding) listBrandOptions().then((b) => setBrands(b || [])).catch(() => {});
  }, [isHolding]);

  const load = useCallback(async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const data = await listPengajuanPerlengkapan({ ...(brandId ? { brand_id: brandId } : {}) });
      setRows(data || []);
    } catch (error) {
      setErrorMessage(error.response?.data?.error || 'Gagal memuat pengajuan perlengkapan.');
    } finally {
      setIsLoading(false);
    }
  }, [brandId]);

  useEffect(() => { load(); }, [load]);

  // Pilihan jadwal diambil dari antrean itu sendiri, sehingga hanya jadwal yang punya pengajuan yang tampil.
  const jadwalOptions = useMemo(() => {
    const seen = new Map();
    rows.forEach((r) => { if (!seen.has(r.schedule_id)) seen.set(r.schedule_id, `${r.jadwal_nama} · ${tanggal(r.berangkat_tanggal)}`); });
    return [{ value: '', label: 'Semua jadwal' }, ...[...seen].map(([value, label]) => ({ value: String(value), label }))];
  }, [rows]);

  // A schedule with no pending request left (all handed over, or another brand picked) falls back to all schedules.
  const jadwalAda = !scheduleId || rows.some((r) => String(r.schedule_id) === scheduleId);
  const jadwalAktif = jadwalAda ? scheduleId : '';
  const tampil = jadwalAktif ? rows.filter((r) => String(r.schedule_id) === jadwalAktif) : rows;

  const serahkan = async (row) => {
    setMemproses(row.pax_id);
    setErrorMessage(null);
    setNotice(null);
    try {
      await serahkanPerlengkapan(row.booking_id, row.pax_id);
      setNotice(`Perlengkapan ${row.nama_jamaah} diserahkan. Jamaah akan diminta mengonfirmasi penerimaan di app.`);
      await load();
    } catch (error) {
      setErrorMessage(error.response?.data?.error || 'Gagal menyerahkan perlengkapan.');
    } finally {
      setMemproses(null);
    }
  };

  const columns = [
    { header: 'Jamaah', key: 'nama_jamaah' },
    { header: 'Keberangkatan', key: 'jadwal_nama' },
    { header: 'Diajukan', key: 'diajukan_at' },
    { header: 'Pengaju', key: 'pengaju_nama' },
    { header: 'Aksi', key: 'aksi', align: 'right' },
  ];

  const renderCell = (r, key) => {
    if (key === 'nama_jamaah') {
      return (
        <div className="min-w-0">
          <div className="font-medium text-neutral-900">{r.nama_jamaah}</div>
          <Link to={`/bookings/${r.booking_id}`} className="font-mono text-xs text-neutral-500 hover:text-neutral-900 underline underline-offset-2 decoration-neutral-300">
            {r.booking_code}
          </Link>
        </div>
      );
    }
    if (key === 'jadwal_nama') {
      return (
        <div className="min-w-0">
          <div className="text-neutral-900">{r.jadwal_nama}</div>
          <div className="text-xs text-neutral-500">{tanggal(r.berangkat_tanggal)}</div>
        </div>
      );
    }
    if (key === 'diajukan_at') {
      return (
        <div className="whitespace-nowrap">
          <div>{waktu(r.diajukan_at)}</div>
          <div className="text-xs text-neutral-500">Ambil di kantor</div>
        </div>
      );
    }
    if (key === 'pengaju_nama') {
      return (
        <div className="min-w-0">
          <div className="text-neutral-900">{r.pengaju_nama || '-'}</div>
          {r.pengaju_hp && <div className="text-xs text-neutral-500">{r.pengaju_hp}</div>}
        </div>
      );
    }
    if (key === 'aksi') {
      return bisaSerahkan ? (
        <Button size="sm" variant="secondary" isLoading={memproses === r.pax_id} disabled={memproses !== null} onClick={() => serahkan(r)}>
          Serahkan
        </Button>
      ) : null;
    }
    return r[key];
  };

  return (
    <div className="space-y-6">
      <PageHeader title="Pengajuan Perlengkapan" />
      {errorMessage && <Alert variant="error" onClose={() => setErrorMessage(null)}>{errorMessage}</Alert>}
      {notice && <Alert variant="success" onClose={() => setNotice(null)}>{notice}</Alert>}
      {isLoading ? (
        <LoadingSpinner variant="card" />
      ) : (
        <DataTable
          columns={columns}
          data={tampil}
          itemsPerPage={20}
          searchPlaceholder="Cari jamaah, kode booking, atau pengaju..."
          emptyMessage="Tidak ada pengajuan perlengkapan yang menunggu diserahkan."
          toolbarActions={
            <div className="flex flex-wrap items-center gap-2">
              <CustomDropdown
                value={jadwalAktif}
                onChange={(v) => setFilter('jadwal', valueOf(v))}
                className="!mb-0 w-64"
                options={jadwalOptions}
              />
              {isHolding && (
                <CustomDropdown
                  value={brandId}
                  onChange={(v) => setFilter('brand', valueOf(v))}
                  className="!mb-0 w-40"
                  options={[{ value: '', label: 'Semua brand' }, ...brands.map((b) => ({ value: String(b.id), label: b.name }))]}
                />
              )}
            </div>
          }
          renderCell={renderCell}
        />
      )}
    </div>
  );
};

export default PengajuanPerlengkapanPage;
