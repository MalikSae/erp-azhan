import React, { useState, useEffect, useMemo } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { Eye, Pencil, Trash2, CalendarDays, Copy, Archive, Send, FilePen } from 'lucide-react';
import PageHeader from 'shared/ui/PageHeader';
import DataTable from 'shared/ui/DataTable';
import Button from 'shared/ui/Button';
import Badge from 'shared/ui/Badge';
import Modal from 'shared/ui/Modal';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import CustomDropdown from 'shared/ui/CustomDropdown';
import ActionMenu from 'shared/ui/ActionMenu';
import { listSchedulesAdmin, deleteSchedule, updateScheduleStatus } from '../api/schedules';
import { listBrands } from '../api/brands';

const STATUS_LABEL = { published: 'Published', draft: 'Draft', archived: 'Archived' };

const STATUS_STRONG = {
  published: '!bg-success-100 !border-success-300 !text-success-800',
  draft: '!border-neutral-400 !text-neutral-800',
  archived: '!bg-neutral-200 !border-neutral-300 !text-neutral-800',
};

const StatusBadge = ({ status }) => {
  if (!STATUS_LABEL[status]) return <span className="text-xs text-neutral-400 font-body">-</span>;
  return <Badge variant={status} hideIcon className={STATUS_STRONG[status]}>{STATUS_LABEL[status]}</Badge>;
};

const formatDate = (dateStr) => {
  if (!dateStr) return '-';
  const d = new Date(dateStr);
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
  return `${d.getDate()} ${months[d.getMonth()]} ${d.getFullYear()}`;
};

const formatCurrency = (amount) => {
  if (amount === undefined || amount === null) return '-';
  return 'Rp ' + amount.toLocaleString('id-ID');
};

const getMinPrice = (row) => {
  const prices = [row.harga_quad, row.harga_triple, row.harga_double].filter(p => p !== null && p !== undefined && p > 0);
  if (prices.length === 0) return 0;
  return Math.min(...prices);
};

const generateRandomCode = (length = 6) => {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
};

const SchedulesPage = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [successMessage, setSuccessMessage] = useState(null);
  const [schedules, setSchedules] = useState([]);
  const [brands, setBrands] = useState([]);
  const [brandsMap, setBrandsMap] = useState({});
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);

  const [deleteConfirmId, setDeleteConfirmId] = useState(null);
  const [confirmCode, setConfirmCode] = useState('');
  const [inputConfirmCode, setInputConfirmCode] = useState('');

  const [activeTab, setActiveTab] = useState('all');
  const [filterBrandId, setFilterBrandId] = useState('');
  const [filterMonth, setFilterMonth] = useState('');

  const today = useMemo(() => {
    const t = new Date();
    t.setHours(0, 0, 0, 0);
    return t;
  }, []);

  const getDaysRemaining = (dateStr) => {
    if (!dateStr) return null;
    const departDate = new Date(dateStr);
    departDate.setHours(0, 0, 0, 0);
    return Math.round((departDate - today) / (1000 * 60 * 60 * 24));
  };

  const isScheduleUrgent = (s) => {
    if (s.status !== 'published') return false;
    const daysRemaining = getDaysRemaining(s.berangkat_tanggal);
    if (daysRemaining === null || daysRemaining < 0 || daysRemaining > 60) return false;
    if (!s.seat_sisa || s.seat_sisa <= 0) return false;
    return true;
  };

  // Paket yang tanggal berangkatnya sudah lewat tetapi belum diarsipkan (MP-09).
  // Tidak diarsipkan otomatis: admin masih perlu mengurus booking setelah berangkat.
  const isScheduleDeparted = (s) => {
    if (s.status === 'archived' || !s.berangkat_tanggal) return false;
    return getDaysRemaining(s.berangkat_tanggal) < 0;
  };

  const fetchSchedules = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const data = await listSchedulesAdmin();
      setSchedules(data || []);

      const brandsData = await listBrands();
      setBrands(brandsData || []);
      const bMap = {};
      (brandsData || []).forEach(b => {
        bMap[b.id] = b;
      });
      setBrandsMap(bMap);
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal memuat data paket.";
      setErrorMessage(msg);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchSchedules();
  }, []);

  // Pesan sukses dari halaman form (simpan/ubah paket); dibersihkan agar tidak muncul lagi saat refresh.
  useEffect(() => {
    if (location.state?.successMessage) {
      setSuccessMessage(location.state.successMessage);
      window.history.replaceState({}, '');
    }
  }, [location.state]);

  const handleDeleteClick = (id) => {
    setDeleteConfirmId(id);
    setConfirmCode(generateRandomCode(6));
    setInputConfirmCode('');
  };

  const handleCloseDeleteModal = () => {
    setDeleteConfirmId(null);
    setConfirmCode('');
    setInputConfirmCode('');
  };

  const changeStatus = async (row, status) => {
    try {
      await updateScheduleStatus(row.id, status);
      fetchSchedules();
    } catch (error) {
      setErrorMessage(error.response?.data?.error || 'Gagal mengubah status paket.');
    }
  };

  const statusItems = (row) => {
    const make = (label, icon, status) => ({ label, icon, tooltip: label, onClick: () => changeStatus(row, status) });
    if (row.status === 'published') return [make('Jadikan Draft', FilePen, 'draft'), make('Arsipkan', Archive, 'archived')];
    if (row.status === 'draft') return [make('Publish', Send, 'published'), make('Arsipkan', Archive, 'archived')];
    if (row.status === 'archived') return [make('Kembalikan ke Draft', FilePen, 'draft')];
    return [];
  };

  const handleConfirmDelete = async () => {
    if (inputConfirmCode !== confirmCode) return;
    try {
      await deleteSchedule(deleteConfirmId);
      handleCloseDeleteModal();
      fetchSchedules();
    } catch (error) {
      handleCloseDeleteModal();
      if (error.response?.status === 409) {
        setErrorMessage(error.response?.data?.error || "Tidak bisa dihapus, data ini sedang digunakan.");
      } else {
        setErrorMessage("Terjadi kesalahan saat menghapus data.");
      }
    }
  };

  const scheduleToDelete = schedules.find(s => s.id === deleteConfirmId);

  const availableMonths = useMemo(() => {
    const map = new Map();
    schedules.forEach(s => {
      if (s.berangkat_tanggal) {
        const d = new Date(s.berangkat_tanggal);
        const val = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
        const formatter = new Intl.DateTimeFormat('id-ID', { month: 'long', year: 'numeric' });
        const label = formatter.format(d);
        map.set(val, label);
      }
    });
    return Array.from(map.entries()).sort((a, b) => a[0].localeCompare(b[0])).map(([value, label]) => ({ value, label }));
  }, [schedules]);

  const counts = useMemo(() => {
    const base = filterBrandId
      ? schedules.filter(s => String(s.brand_id) === filterBrandId)
      : schedules;

    return {
      all: base.length,
      urgent: base.filter(isScheduleUrgent).length,
      published: base.filter(s => s.status === 'published').length,
      draft: base.filter(s => s.status === 'draft').length,
      archived: base.filter(s => s.status === 'archived').length,
      departed: base.filter(isScheduleDeparted).length,
    };
  }, [schedules, filterBrandId, today]);

  const rows = useMemo(() => {
    return schedules
      .filter(s => {
        if (activeTab === 'urgent' && !isScheduleUrgent(s)) return false;
        if (activeTab === 'published' && s.status !== 'published') return false;
        if (activeTab === 'draft' && s.status !== 'draft') return false;
        if (activeTab === 'archived' && s.status !== 'archived') return false;
        if (activeTab === 'departed' && !isScheduleDeparted(s)) return false;

        if (filterBrandId && String(s.brand_id) !== filterBrandId) return false;

        if (filterMonth) {
          if (!s.berangkat_tanggal) return false;
          const d = new Date(s.berangkat_tanggal);
          const val = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
          if (val !== filterMonth) return false;
        }
        return true;
      })
      // Kolom bantu pencarian: brand, maskapai, dan kategori ikut bisa dicari.
      .map(s => ({
        ...s,
        _cari: [brandsMap[s.brand_id]?.name, s.maskapai?.name, s.category?.name].filter(Boolean).join(' '),
      }))
      // Urutan bawaan: yang paling dekat berangkat di atas; yang sudah lewat di bawah.
      .sort((a, b) => {
        const da = getDaysRemaining(a.berangkat_tanggal);
        const db = getDaysRemaining(b.berangkat_tanggal);
        if (da === null && db === null) return 0;
        if (da === null) return 1;
        if (db === null) return -1;
        const pastA = da < 0;
        const pastB = db < 0;
        if (pastA !== pastB) return pastA ? 1 : -1;
        return pastA ? db - da : da - db;
      });
  }, [schedules, activeTab, filterBrandId, filterMonth, brandsMap, today]);

  const tabs = [
    { id: 'all', label: 'Semua', count: counts.all },
    { id: 'urgent', label: 'Mendesak', count: counts.urgent },
    { id: 'published', label: 'Published', count: counts.published },
    { id: 'draft', label: 'Draft', count: counts.draft },
    { id: 'archived', label: 'Archived', count: counts.archived },
    { id: 'departed', label: 'Perlu Diarsipkan', count: counts.departed },
  ];

  const columns = [
    { header: 'Paket', key: 'jadwal_nama' },
    {
      header: 'Keberangkatan',
      key: 'berangkat_tanggal',
      sortable: true,
      sortFn: (a, b) => String(a.berangkat_tanggal || '').localeCompare(String(b.berangkat_tanggal || '')),
    },
    {
      header: 'Terisi',
      key: 'seat_sisa',
      sortable: true,
      sortFn: (a, b) => ((a.seat_total || 0) - (a.seat_sisa || 0)) - ((b.seat_total || 0) - (b.seat_sisa || 0)),
    },
    {
      header: 'Harga Mulai',
      key: 'harga',
      sortable: true,
      sortFn: (a, b) => getMinPrice(a) - getMinPrice(b),
    },
    { header: 'Status', key: 'status' },
    { header: 'Aksi', key: 'aksi' },
  ];

  const renderCell = (row, key) => {
    if (key === 'jadwal_nama') {
      const brand = brandsMap[row.brand_id];
      const sub = [row.maskapai?.name, row.is_direct_flight ? 'Direct' : null, row.category?.name].filter(Boolean).join(' · ');
      return (
        <div className="min-w-0">
          <button
            type="button"
            onClick={(e) => { e.stopPropagation(); navigate(`/schedules/${row.id}`); }}
            className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors"
            title="Lihat Detail Paket"
          >
            {row.jadwal_nama}
          </button>
          <div className="mt-0.5 flex items-center gap-1.5 text-[11px] font-body text-neutral-500 min-w-0">
            {brand && (
              <span className="inline-flex items-center gap-1.5 shrink-0 font-medium text-neutral-700">
                <span className="w-2 h-2 rounded-full" style={{ backgroundColor: brand.primary_color || '#9CA3AF' }} />
                {brand.name}
              </span>
            )}
            {sub && <span className="truncate text-neutral-400">{brand ? '· ' : ''}{sub}</span>}
          </div>
        </div>
      );
    }

    if (key === 'berangkat_tanggal') {
      const days = getDaysRemaining(row.berangkat_tanggal);
      return (
        <div>
          <div className="flex items-center gap-1.5 text-[13px] leading-snug font-body text-neutral-700 whitespace-nowrap">
            <CalendarDays className="w-3.5 h-3.5 text-neutral-400" strokeWidth={1.75} />
            {formatDate(row.berangkat_tanggal)}
          </div>
          {isScheduleDeparted(row) ? (
            <div className="mt-0.5 text-[11px] font-body text-neutral-400">Sudah berangkat</div>
          ) : days !== null && days >= 0 ? (
            <div
              className={`mt-0.5 text-[11px] font-body ${isScheduleUrgent(row) ? 'text-danger-600 font-medium' : 'text-neutral-400'}`}
              title={isScheduleUrgent(row) ? 'Mendekati keberangkatan & belum penuh' : undefined}
            >
              {days} hari lagi
            </div>
          ) : null}
        </div>
      );
    }

    if (key === 'status') return <StatusBadge status={row.status} />;

    if (key === 'seat_sisa') {
      const total = Number(row.seat_total) || 0;
      const sisa = row.seat_sisa !== null && row.seat_sisa !== undefined ? Number(row.seat_sisa) : null;
      if (sisa === null) return <span className="text-xs text-neutral-400 font-body">-</span>;
      if (sisa <= 0) {
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded-md text-[11px] font-bold font-heading tracking-wide uppercase bg-neutral-900 text-white">
            Penuh
          </span>
        );
      }
      const filled = Math.max(0, total - sisa);
      const urgent = isScheduleUrgent(row);
      return (
        <div className="min-w-[120px]" title={urgent ? 'Mendekati keberangkatan dan masih banyak kursi kosong' : undefined}>
          <div className="text-[13px] font-body text-neutral-700 leading-snug whitespace-nowrap">
            <span className="font-semibold text-neutral-900">{filled}</span> dari {total} terisi
          </div>
          <div className={`mt-0.5 text-[11px] font-body ${urgent ? 'text-danger-600 font-medium' : 'text-neutral-400'}`}>
            {urgent ? `Mendesak · ${sisa} kursi kosong` : `${sisa} kursi kosong`}
          </div>
        </div>
      );
    }

    if (key === 'harga') {
      const minPrice = getMinPrice(row);
      if (minPrice <= 0) return <span className="text-xs text-neutral-400 font-body">-</span>;
      return <span className="text-[13px] font-semibold text-neutral-900 font-body whitespace-nowrap">{formatCurrency(minPrice)}</span>;
    }

    if (key === 'aksi') {
      const hasBookings = (row.booking_count || 0) > 0;
      const menuItems = [
        { label: 'Lihat', icon: Eye, tooltip: 'Lihat Detail Paket', onClick: () => navigate(`/schedules/${row.id}`) },
        { label: 'Edit', icon: Pencil, tooltip: 'Edit Paket', onClick: () => navigate(`/schedules/${row.id}/edit`) },
        { label: 'Duplikat', icon: Copy, tooltip: 'Buat paket baru dari paket ini', onClick: () => navigate(`/schedules/new?from=${row.id}`) },
        ...statusItems(row),
        {
          label: 'Hapus',
          icon: Trash2,
          danger: true,
          tooltip: 'Hapus Paket',
          disabled: hasBookings,
          disabledTooltip: `Tidak bisa dihapus, memiliki ${row.booking_count} booking jamaah`,
          onClick: () => handleDeleteClick(row.id),
        },
      ];
      return <ActionMenu items={menuItems} align="right" />;
    }

    return row[key];
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Kelola Paket"
        actionLabel="+ Tambah Paket"
        onAction={() => navigate('/schedules/new')}
      />

      {successMessage && (
        <Alert variant="success" onClose={() => setSuccessMessage(null)}>{successMessage}</Alert>
      )}

      {errorMessage && (
        <Alert variant="error" onClose={() => setErrorMessage(null)}>{errorMessage}</Alert>
      )}

      {isLoading ? (
        <div className="flex justify-center p-8 bg-white rounded-2xl border border-neutral-200 shadow-card"><LoadingSpinner /></div>
      ) : (
        <DataTable
          columns={columns}
          data={rows}
          itemsPerPage={15}
          onRowClick={(row) => navigate(`/schedules/${row.id}`)}
          emptyMessage={activeTab === 'urgent' ? 'Tidak ada paket mendesak saat ini.' : 'Belum ada paket. Klik "+ Tambah Paket" untuk menambahkan.'}
          searchPlaceholder="Cari nama paket, brand, atau maskapai..."
          toolbarActions={
            <div className="flex flex-wrap items-center gap-2">
              <CustomDropdown
                value={activeTab}
                onChange={(val) => setActiveTab(val)}
                className="!mb-0 w-48"
                placeholder="Semua Status"
                options={tabs.map(tab => ({ value: tab.id, label: `${tab.label} (${tab.count})` }))}
              />
              {brands.length > 1 && (
                <CustomDropdown
                  value={filterBrandId}
                  onChange={(val) => setFilterBrandId(val)}
                  className="!mb-0 w-40"
                  placeholder="Semua Brand"
                  options={[
                    { value: '', label: 'Semua Brand' },
                    ...brands.map(b => ({ value: b.id.toString(), label: b.name })),
                  ]}
                />
              )}
              <CustomDropdown
                value={filterMonth}
                onChange={(val) => setFilterMonth(val)}
                className="!mb-0 w-40"
                placeholder="Semua Bulan"
                options={[
                  { value: '', label: 'Semua Bulan' },
                  ...availableMonths.map(m => ({ value: m.value, label: m.label })),
                ]}
              />
            </div>
          }
          renderCell={renderCell}
        />
      )}

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteConfirmId !== null}
        onClose={handleCloseDeleteModal}
        title="Hapus Paket?"
        size="sm"
        footer={
          <>
            <Button variant="ghost" onClick={handleCloseDeleteModal}>Batal</Button>
            <Button
              variant="danger"
              onClick={handleConfirmDelete}
              disabled={!confirmCode || inputConfirmCode !== confirmCode}
            >
              Hapus
            </Button>
          </>
        }
      >
        <div className="space-y-4 font-body">
          <p className="text-neutral-600 text-sm">
            Yakin ingin menghapus <strong>{scheduleToDelete?.jadwal_nama}</strong>? Tindakan ini tidak bisa dibatalkan.
          </p>

          <div className="bg-neutral-100 border border-neutral-200 rounded-md p-3 text-center">
            <span className="text-xs text-neutral-500 block mb-1">Kode Konfirmasi:</span>
            <span className="font-mono text-xl font-bold tracking-widest text-neutral-800 select-all">
              {confirmCode}
            </span>
          </div>

          <div>
            <label className="block text-xs font-medium text-neutral-700 mb-1">
              Ketik ulang kode di atas:
            </label>
            <input
              type="text"
              value={inputConfirmCode}
              onChange={(e) => setInputConfirmCode(e.target.value)}
              placeholder="Ketik kode di atas untuk konfirmasi"
              className="w-full px-3 py-2 text-sm border border-neutral-300 rounded-md focus:outline-none focus:ring-1 focus:ring-danger-500 focus:border-danger-500 font-mono tracking-wider text-neutral-900 bg-white"
              autoFocus
            />
            <p className="text-xs text-neutral-500 mt-1">
              Ketik persis sama termasuk huruf besar/kecil
            </p>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default SchedulesPage;
