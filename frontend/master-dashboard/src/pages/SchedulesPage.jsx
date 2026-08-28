import React, { useState, useEffect, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { Files, Check, Archive, Minus, Zap } from 'lucide-react';
import PageHeader from '../components/ui/PageHeader';
import Card from '../components/ui/Card';
import DataTable from '../components/ui/DataTable';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import Badge from '../components/ui/Badge';
import CustomDropdown from '../components/ui/CustomDropdown';
import { listSchedulesAdmin, deleteSchedule } from '../api/schedules';
import { listBrands } from '../api/brands';
import UrgentPackagesBanner from '../components/UrgentPackagesBanner';

const BrandCell = ({ brand }) => {
  const [imageError, setImageError] = useState(false);
  
  if (!brand) return <span>-</span>;
  
  const showInitial = !brand.icon_url || imageError;
  
  return (
    <div className="flex items-center" title={brand.name}>
      {showInitial ? (
        <div className="w-7 h-7 shrink-0 rounded-full bg-neutral-200 border border-neutral-300 flex items-center justify-center text-xs font-bold text-neutral-700 uppercase font-heading">
          {brand.name.charAt(0)}
        </div>
      ) : (
        <img 
          src={brand.icon_url.startsWith('http') ? brand.icon_url : `${import.meta.env.VITE_API_BASE_URL}${brand.icon_url.startsWith('/') ? '' : '/'}${brand.icon_url}`} 
          alt={brand.name} 
          className="w-7 h-7 shrink-0 rounded-full object-cover bg-neutral-100 border border-neutral-200" 
          onError={() => setImageError(true)}
        />
      )}
    </div>
  );
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

  const isScheduleUrgent = (s) => {
    if (!s.is_ticket_confirmed) return false;
    if (s.status === 'archived') return false;
    if (!s.berangkat_tanggal) return false;
    const departDate = new Date(s.berangkat_tanggal);
    departDate.setHours(0, 0, 0, 0);
    const daysRemaining = Math.round((departDate - today) / (1000 * 60 * 60 * 24));
    if (daysRemaining < 0 || daysRemaining > 60) return false;
    if (!s.seat_sisa || s.seat_sisa <= 0) return false;
    return true;
  };

  const getDaysRemaining = (dateStr) => {
    if (!dateStr) return null;
    const departDate = new Date(dateStr);
    departDate.setHours(0, 0, 0, 0);
    return Math.round((departDate - today) / (1000 * 60 * 60 * 24));
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
    };
  }, [schedules, filterBrandId, today]);

  const filteredSchedules = useMemo(() => {
    return schedules.filter(s => {
      // 1. Tab Filter
      if (activeTab === 'urgent') {
        if (!isScheduleUrgent(s)) return false;
      } else if (activeTab === 'published') {
        if (s.status !== 'published') return false;
      } else if (activeTab === 'draft') {
        if (s.status !== 'draft') return false;
      } else if (activeTab === 'archived') {
        if (s.status !== 'archived') return false;
      }

      // 2. Brand Filter
      if (filterBrandId && String(s.brand_id) !== filterBrandId) {
        return false;
      }

      // 3. Month Filter
      if (filterMonth) {
        if (!s.berangkat_tanggal) return false;
        const d = new Date(s.berangkat_tanggal);
        const val = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
        if (val !== filterMonth) return false;
      }

      return true;
    });
  }, [schedules, activeTab, filterBrandId, filterMonth, today]);

  const tabs = [
    { id: 'all', label: 'Semua Paket', count: counts.all },
    { id: 'urgent', label: '🚨 Mendesak (<60 Hari)', count: counts.urgent, isUrgent: true },
    { id: 'published', label: 'Published', count: counts.published },
    { id: 'draft', label: 'Draft', count: counts.draft },
    { id: 'archived', label: 'Archived', count: counts.archived },
  ];

  const columns = [
    { header: 'Brand', key: 'brand' }
  ];
  columns.push(
    { header: 'Nama Paket', key: 'jadwal_nama' },
    { header: 'Publish', key: 'status', align: 'center' },
    { 
      header: 'Tiket', 
      key: 'tiket',
      align: 'center',
      sortable: true,
      sortFn: (a, b) => {
        const valA = a.is_ticket_confirmed ? 1 : 0;
        const valB = b.is_ticket_confirmed ? 1 : 0;
        return valA - valB;
      }
    },
    {
      header: 'Promo',
      key: 'promo',
      align: 'center',
      sortable: true,
      sortFn: (a, b) => {
        const valA = a.is_promo ? 1 : 0;
        const valB = b.is_promo ? 1 : 0;
        return valA - valB;
      }
    },
    { 
      header: 'Berangkat', 
      key: 'berangkat_tanggal',
      sortable: true,
      sortFn: (a, b) => {
        const tA = a.berangkat_tanggal ? new Date(a.berangkat_tanggal).getTime() : 0;
        const tB = b.berangkat_tanggal ? new Date(b.berangkat_tanggal).getTime() : 0;
        return tA - tB;
      }
    },

    { 
      header: 'Sisa Seat', 
      key: 'seat_sisa',
      sortable: true,
      sortFn: (a, b) => (a.seat_sisa || 0) - (b.seat_sisa || 0)
    },
    { 
      header: 'Harga Mulai', 
      key: 'harga',
      sortable: true,
      sortFn: (a, b) => getMinPrice(a) - getMinPrice(b)
    },
    { header: 'Aksi', key: 'aksi' }
  );

  return (
    <div className="space-y-6">
      <PageHeader 
        title="Kelola Paket" 
        actionLabel="+ Tambah Paket" 
        onAction={() => navigate('/schedules/new')}
      />

      {errorMessage && (
        <Alert variant="error">{errorMessage}</Alert>
      )}

      {/* Quick Filter Tabs (Option 3) */}
      <div className="flex flex-wrap items-center gap-2 border-b border-neutral-200 pb-3">
        {tabs.map((tab) => {
          const isActive = activeTab === tab.id;
          const isUrgentTab = tab.id === 'urgent';
          const hasUrgentCount = isUrgentTab && tab.count > 0;

          let btnClass = 'bg-white text-neutral-600 hover:bg-neutral-50 border border-neutral-200/90';
          let badgeClass = 'bg-neutral-100 text-neutral-600';

          if (isActive) {
            btnClass = isUrgentTab ? 'bg-danger-600 text-white shadow-2xs font-bold' : 'bg-sidebar-bg text-white shadow-2xs font-bold';
            badgeClass = 'bg-white/20 text-white';
          } else if (hasUrgentCount) {
            btnClass = 'bg-danger-50 text-danger-700 hover:bg-danger-100 border border-danger-200';
            badgeClass = 'bg-danger-200 text-danger-800 font-bold';
          }

          return (
            <button
              key={tab.id}
              type="button"
              onClick={() => setActiveTab(tab.id)}
              className={`inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full text-xs font-semibold font-heading transition-all ${btnClass}`}
            >
              <span>{tab.label}</span>
              <span className={`px-2 py-0.5 rounded-full text-[11px] ${badgeClass}`}>
                {tab.count}
              </span>
            </button>
          );
        })}
      </div>

      {isLoading ? (
        <div className="flex justify-center p-8 bg-white rounded-lg border border-neutral-200 shadow-sm">
          <LoadingSpinner />
        </div>
      ) : (
        <div className="space-y-4">
          <DataTable
            columns={columns}
            data={filteredSchedules}
            itemsPerPage={15}
            onRowClick={(row) => navigate(`/schedules/${row.id}`)}
            emptyMessage={activeTab === 'urgent' ? 'Tidak ada paket mendesak saat ini.' : 'Belum ada paket. Klik "+ Tambah Paket" untuk menambahkan.'}
            toolbarActions={
              <div className="flex flex-wrap items-center gap-2">
                <CustomDropdown 
                  value={filterBrandId}
                  onChange={(val) => setFilterBrandId(val)}
                  className="!mb-0 w-40"
                  placeholder="Semua Brand"
                  options={[
                    { value: '', label: 'Semua Brand' },
                    ...brands.map(b => ({ value: b.id.toString(), label: b.name }))
                  ]}
                />
                <CustomDropdown 
                  value={filterMonth}
                  onChange={(val) => setFilterMonth(val)}
                  className="!mb-0 w-40"
                  placeholder="Semua Bulan"
                  options={[
                    { value: '', label: 'Semua Bulan' },
                    ...availableMonths.map(m => ({ value: m.value, label: m.label }))
                  ]}
                />
              </div>
            }
            renderCell={(row, key) => {
              if (key === 'brand') {
                return <BrandCell brand={brandsMap[row.brand_id]} />;
              }
            if (key === 'status') {
              if (row.status === 'published') {
                return (
                  <div className="flex items-center justify-center">
                    <div title="Published" className="w-5 h-5 rounded-full bg-success-500 text-white flex items-center justify-center shadow-2xs">
                      <Check size={12} strokeWidth={3} />
                    </div>
                  </div>
                );
              }
              if (row.status === 'draft') {
                return (
                  <div className="flex items-center justify-center">
                    <div title="Draft" className="flex items-center justify-center text-neutral-600">
                      <Files size={18} />
                    </div>
                  </div>
                );
              }
              if (row.status === 'archived') {
                return (
                  <div className="flex items-center justify-center">
                    <div title="Archived" className="flex items-center justify-center text-neutral-400">
                      <Archive size={18} />
                    </div>
                  </div>
                );
              }
              return <div className="text-center text-xs text-neutral-400">-</div>;
            }
            if (key === 'tiket') {
              return (
                <div className="flex items-center justify-center">
                  {row.is_ticket_confirmed ? (
                    <div title="Tiket Confirmed" className="w-5 h-5 rounded-full bg-success-500 text-white flex items-center justify-center shadow-2xs">
                      <Check size={12} strokeWidth={3} />
                    </div>
                  ) : (
                    <div title="Tiket Belum Confirmed" className="w-5 h-5 rounded-full bg-neutral-200 text-neutral-400 flex items-center justify-center shadow-2xs">
                      <Minus size={12} strokeWidth={3} />
                    </div>
                  )}
                </div>
              );
            }
            if (key === 'promo') {
              return (
                <div className="flex items-center justify-center">
                  {row.is_promo ? (
                    <div title="Promo Aktif" className="w-5 h-5 rounded-full bg-warning-500 text-white flex items-center justify-center shadow-2xs">
                      <Zap size={11} className="fill-white" />
                    </div>
                  ) : (
                    <div title="Tidak Ada Promo" className="w-5 h-5 rounded-full bg-neutral-200 text-neutral-400 flex items-center justify-center shadow-2xs">
                      <Minus size={12} strokeWidth={3} />
                    </div>
                  )}
                </div>
              );
            }
            if (key === 'berangkat_tanggal') {
              const days = getDaysRemaining(row.berangkat_tanggal);
              const isUrgent = isScheduleUrgent(row);
              return (
                <div className="flex items-center gap-2">
                  <span className="font-body text-neutral-900">{formatDate(row.berangkat_tanggal)}</span>
                  {isUrgent && days !== null && (
                    <span 
                      className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-bold text-danger-700 bg-danger-50 border border-danger-200"
                      title="Mendekati keberangkatan & belum penuh"
                    >
                      {days} hr lagi
                    </span>
                  )}
                </div>
              );
            }

            if (key === 'seat_sisa') {
              if (row.seat_sisa === 0) {
                return (
                  <div className="flex items-center">
                    <span className="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-semibold font-heading tracking-wide bg-success-600 text-white shadow-2xs">
                      Full Booked
                    </span>
                  </div>
                );
              }
              const isUrgent = isScheduleUrgent(row);
              const filled = (row.seat_total || 0) - (row.seat_sisa || 0);
              const percent = row.seat_total > 0 ? (filled / row.seat_total) * 100 : 0;
              return (
                <div className="flex flex-col">
                  <div className="flex items-center gap-1 font-body">
                    <span className={`font-semibold ${isUrgent ? 'text-danger-700 font-bold' : 'text-neutral-900'}`}>
                      {row.seat_sisa ?? '-'}
                    </span>
                    <span className="text-neutral-400 text-xs">/ {row.seat_total ?? '-'} pax</span>
                  </div>
                  {isUrgent && row.seat_total > 0 && (
                    <div className="w-20 h-1.5 bg-neutral-100 rounded-full overflow-hidden mt-1 border border-neutral-200" title={`Terisi ${filled}/${row.seat_total} pax`}>
                      <div 
                        className="h-full bg-danger-500 rounded-full" 
                        style={{ width: `${Math.max(0, Math.min(100, percent))}%` }} 
                      />
                    </div>
                  )}
                </div>
              );
            }

            if (key === 'harga') {
               const minPrice = getMinPrice(row);
               return minPrice > 0 ? formatCurrency(minPrice) : '-';
            }
            if (key === 'aksi') {
              const hasBookings = (row.booking_count || 0) > 0;
              return (
                <div className="flex gap-2 items-center">
                  <Button 
                    variant="ghost" 
                    size="sm" 
                    onClick={(e) => {
                      e.stopPropagation();
                      navigate(`/schedules/${row.id}/edit`);
                    }}
                    className="!px-2"
                    title="Edit"
                  >
                    Edit
                  </Button>
                  {hasBookings ? (
                    <button 
                      type="button"
                      disabled
                      onClick={(e) => e.stopPropagation()}
                      title={`Tidak bisa dihapus, memiliki ${row.booking_count} booking jamaah`}
                      className="text-neutral-300 opacity-40 cursor-not-allowed ml-1"
                    >
                      <svg className="w-5 h-5 pointer-events-none" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  ) : (
                    <button 
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleDeleteClick(row.id);
                      }} 
                      title="Hapus"
                      className="text-neutral-400 hover:text-danger-600 transition-colors ml-1 cursor-pointer"
                    >
                      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  )}
                </div>
              );
            }
            return row[key];
          }}
          />
        </div>
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
