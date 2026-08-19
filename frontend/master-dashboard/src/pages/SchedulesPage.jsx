import React, { useState, useEffect, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
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
  
  if (!brand) return null;
  
  const showInitial = !brand.icon_url || imageError;
  
  return (
    <div className="flex items-center gap-2">
      {showInitial ? (
        <div className="w-6 h-6 shrink-0 rounded-full bg-neutral-200 flex items-center justify-center text-xs font-bold text-neutral-600 uppercase">
          {brand.name.charAt(0)}
        </div>
      ) : (
        <img 
          src={brand.icon_url.startsWith('http') ? brand.icon_url : `${import.meta.env.VITE_API_BASE_URL}${brand.icon_url.startsWith('/') ? '' : '/'}${brand.icon_url}`} 
          alt={brand.name} 
          className="w-6 h-6 shrink-0 rounded-full object-cover bg-neutral-100" 
          onError={() => setImageError(true)}
        />
      )}
      <span>{brand.name}</span>
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

const SchedulesPage = () => {
  const navigate = useNavigate();
  const [schedules, setSchedules] = useState([]);
  const [brands, setBrands] = useState([]);
  const [brandsMap, setBrandsMap] = useState({});
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  
  const [deleteConfirmId, setDeleteConfirmId] = useState(null);

  const [filterStatus, setFilterStatus] = useState('');
  const [filterBrandId, setFilterBrandId] = useState('');
  const [filterMonth, setFilterMonth] = useState('');

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
  };

  const handleConfirmDelete = async () => {
    try {
      await deleteSchedule(deleteConfirmId);
      setDeleteConfirmId(null);
      fetchSchedules();
    } catch (error) {
      setDeleteConfirmId(null);
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

  const filteredSchedules = schedules.filter(s => {
    if (filterStatus) {
      if (s.status !== filterStatus) return false;
    }
    if (filterBrandId) {
      if (String(s.brand_id) !== filterBrandId) return false;
    }
    if (filterMonth) {
      if (!s.berangkat_tanggal) return false;
      const d = new Date(s.berangkat_tanggal);
      const val = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
      if (val !== filterMonth) return false;
    }
    return true;
  });

  const urgentPackages = useMemo(() => {
    const today = new Date();
    today.setHours(0, 0, 0, 0);

    const urgent = schedules.filter(s => {
      if (!s.is_ticket_confirmed) return false;
      if (s.status === 'archived') return false;
      if (filterBrandId && String(s.brand_id) !== filterBrandId) return false;

      if (!s.berangkat_tanggal) return false;
      const departDate = new Date(s.berangkat_tanggal);
      departDate.setHours(0, 0, 0, 0);
      const daysRemaining = Math.round((departDate - today) / (1000 * 60 * 60 * 24));
      
      if (daysRemaining < 0 || daysRemaining > 60) return false;
      if (!s.seat_sisa || s.seat_sisa <= 0) return false;

      return true;
    }).map(s => {
      const departDate = new Date(s.berangkat_tanggal);
      departDate.setHours(0, 0, 0, 0);
      const daysRemaining = Math.round((departDate - today) / (1000 * 60 * 60 * 24));

      return {
        id: s.id,
        jadwal_nama: s.jadwal_nama,
        brand_name: brandsMap[s.brand_id]?.name || 'Unknown',
        daysRemaining,
        seat_sisa: s.seat_sisa,
        seat_total: s.seat_total
      };
    });

    return urgent.sort((a, b) => a.daysRemaining - b.daysRemaining);
  }, [schedules, brandsMap, filterBrandId]);

  const columns = [
    { header: 'Brand', key: 'brand' }
  ];
  columns.push(
    { header: 'Nama Paket', key: 'jadwal_nama' },
    { header: 'Status', key: 'status' },
    { 
      header: 'Tiket', 
      key: 'tiket',
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

      <UrgentPackagesBanner packages={urgentPackages} />

      {errorMessage && (
        <Alert variant="error">{errorMessage}</Alert>
      )}

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
            emptyMessage='Belum ada paket. Klik "+ Tambah Paket" untuk menambahkan.'
            toolbarActions={
              <div className="flex flex-col sm:flex-row gap-2 w-full">
                <CustomDropdown 
                  value={filterStatus}
                  onChange={(val) => setFilterStatus(val)}
                  className="!mb-0 w-full sm:w-40"
                  placeholder="Semua Status"
                  options={[
                    { value: '', label: 'Semua Status' },
                    { value: 'draft', label: 'Draft' },
                    { value: 'published', label: 'Published' },
                    { value: 'archived', label: 'Archived' }
                  ]}
                />
                <CustomDropdown 
                  value={filterBrandId}
                  onChange={(val) => setFilterBrandId(val)}
                  className="!mb-0 w-full sm:w-40"
                  placeholder="Semua Brand"
                  options={[
                    { value: '', label: 'Semua Brand' },
                    ...brands.map(b => ({ value: b.id.toString(), label: b.name }))
                  ]}
                />
                <CustomDropdown 
                  value={filterMonth}
                  onChange={(val) => setFilterMonth(val)}
                  className="!mb-0 w-full sm:w-40"
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
              const label = row.status.charAt(0).toUpperCase() + row.status.slice(1);
              return <Badge variant={row.status}>{label}</Badge>;
            }
            if (key === 'tiket') {
              return row.is_ticket_confirmed ? (
                <div title="Confirmed" className="flex items-center text-success-500">
                  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                  </svg>
                </div>
              ) : (
                <div title="Belum Confirmed" className="flex items-center text-neutral-300">
                  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM7 9a1 1 0 000 2h6a1 1 0 100-2H7z" clipRule="evenodd" />
                  </svg>
                </div>
              );
            }
            if (key === 'promo') {
              return row.is_promo ? (
                <div title="Promo Aktif" className="flex items-center text-warning-500">
                  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M11.983 1.907a.75.75 0 0 0-1.292-.656l-8.5 9.5A.75.75 0 0 0 2.75 12h6.572l-1.305 6.093a.75.75 0 0 0 1.292.656l8.5-9.5A.75.75 0 0 0 17.25 8h-6.572l1.305-6.093Z" />
                  </svg>
                </div>
              ) : (
                <div title="Tidak Ada Promo" className="flex items-center text-neutral-300">
                  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M10 18a8 8 0 1 0 0-16 8 8 0 0 0 0 16ZM6.75 9.25a.75.75 0 0 0 0 1.5h6.5a.75.75 0 0 0 0-1.5h-6.5Z" clipRule="evenodd" />
                  </svg>
                </div>
              );
            }
            if (key === 'berangkat_tanggal') return formatDate(row.berangkat_tanggal);

            if (key === 'harga') {
               const minPrice = getMinPrice(row);
               return minPrice > 0 ? formatCurrency(minPrice) : '-';
            }
            if (key === 'aksi') {
              return (
                <div className="flex gap-2 items-center">
                  <Button 
                    variant="ghost" 
                    size="sm" 
                    onClick={() => navigate(`/schedules/${row.id}/edit`)}
                    className="!px-2"
                    title="Edit"
                  >
                    Edit
                  </Button>
                  <button 
                    onClick={() => handleDeleteClick(row.id)} 
                    title="Hapus"
                    className="text-neutral-400 hover:text-danger-600 transition-colors ml-1"
                  >
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
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
        onClose={() => setDeleteConfirmId(null)}
        title="Hapus Paket?"
        size="sm"
        footer={
          <>
            <Button variant="ghost" onClick={() => setDeleteConfirmId(null)}>Batal</Button>
            <Button variant="danger" onClick={handleConfirmDelete}>Hapus</Button>
          </>
        }
      >
        <p className="text-neutral-600 font-body">
          Yakin ingin menghapus <strong>{scheduleToDelete?.jadwal_nama}</strong>? Tindakan ini tidak bisa dibatalkan.
        </p>
      </Modal>
    </div>
  );
};

export default SchedulesPage;
