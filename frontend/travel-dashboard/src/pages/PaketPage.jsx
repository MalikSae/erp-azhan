import React, { useState, useEffect, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import PageHeader from '../components/ui/PageHeader';
import DataTable from '../components/ui/DataTable';
import Button from '../components/ui/Button';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import Badge from '../components/ui/Badge';
import CustomDropdown from '../components/ui/CustomDropdown';
import { listSchedulesAdmin } from '../api/schedules';
import UrgentPackagesBanner from '../components/UrgentPackagesBanner';
import { Eye } from 'lucide-react';

const formatDate = (dateStr) => {
  if (!dateStr) return '-';
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return dateStr;
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

const PaketPage = () => {
  const navigate = useNavigate();
  const [schedules, setSchedules] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  
  const [filterStatus, setFilterStatus] = useState('');
  const [filterMonth, setFilterMonth] = useState('');
  const [filterMaskapai, setFilterMaskapai] = useState('');

  const fetchSchedules = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const data = await listSchedulesAdmin();
      setSchedules(data || []);
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

  const availableMaskapai = useMemo(() => {
    const map = new Map();
    schedules.forEach(s => {
      if (s.maskapai && s.maskapai.id) {
        map.set(String(s.maskapai.id), s.maskapai.name);
      }
    });
    return Array.from(map.entries())
      .map(([value, label]) => ({ value, label }))
      .sort((a, b) => a.label.localeCompare(b.label));
  }, [schedules]);

  const availableMonths = useMemo(() => {
    const map = new Map();
    schedules.forEach(s => {
      if (s.berangkat_tanggal) {
        const d = new Date(s.berangkat_tanggal);
        if (!isNaN(d.getTime())) {
          const val = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
          const formatter = new Intl.DateTimeFormat('id-ID', { month: 'long', year: 'numeric' });
          const label = formatter.format(d);
          map.set(val, label);
        }
      }
    });
    return Array.from(map.entries()).sort((a, b) => a[0].localeCompare(b[0])).map(([value, label]) => ({ value, label }));
  }, [schedules]);

  const filteredSchedules = schedules.filter(s => {
    if (filterMaskapai) {
      if (!s.maskapai || String(s.maskapai.id) !== String(filterMaskapai)) return false;
    }
    if (filterStatus) {
      if (s.status !== filterStatus) return false;
    }
    if (filterMonth) {
      if (!s.berangkat_tanggal) return false;
      const d = new Date(s.berangkat_tanggal);
      if (isNaN(d.getTime())) return false;
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
        daysRemaining,
        seat_sisa: s.seat_sisa,
        seat_total: s.seat_total,
        rawSchedule: s
      };
    });

    return urgent.sort((a, b) => a.daysRemaining - b.daysRemaining);
  }, [schedules]);

  const handleOpenDetail = (pkgOrSchedule) => {
    const id = pkgOrSchedule.id || pkgOrSchedule.rawSchedule?.id;
    if (id) {
      navigate(`/paket/${id}`);
    }
  };

  const columns = [
    { 
      header: 'Nama Paket', 
      key: 'jadwal_nama',
      accessor: (row) => (
        <button
          type="button"
          onClick={() => navigate(`/paket/${row.id}`)}
          className="font-semibold text-neutral-900 hover:text-primary-600 hover:underline text-left transition-colors"
          title="Lihat Detail Paket"
        >
          {row.jadwal_nama}
        </button>
      )
    },
    { 
      header: 'Maskapai', 
      key: 'maskapai',
      accessor: (row) => {
        const logo = row.maskapai?.logo_url;
        const name = row.maskapai?.name;
        if (logo) {
          const logoSrc = logo.startsWith('http') ? logo : `${import.meta.env.VITE_API_BASE_URL || 'http://localhost:9090'}${logo}`;
          return (
            <div className="flex items-center" title={name || 'Maskapai'}>
              <div className="w-10 h-7 rounded border border-neutral-200 bg-white p-0.5 flex items-center justify-center overflow-hidden shadow-2xs">
                <img 
                  src={logoSrc} 
                  alt={name || 'Logo Maskapai'} 
                  className="max-w-full max-h-full object-contain"
                />
              </div>
            </div>
          );
        }
        return <span className="text-xs font-semibold text-neutral-700">{name || '-'}</span>;
      }
    },
    { 
      header: 'Status', 
      key: 'status',
      accessor: (row) => {
        const label = row.status.charAt(0).toUpperCase() + row.status.slice(1);
        return <Badge variant={row.status}>{label}</Badge>;
      }
    },
    { 
      header: 'Tiket', 
      key: 'tiket',
      sortable: true,
      sortFn: (a, b) => {
        const valA = a.is_ticket_confirmed ? 1 : 0;
        const valB = b.is_ticket_confirmed ? 1 : 0;
        return valA - valB;
      },
      accessor: (row) => row.is_ticket_confirmed ? (
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
      )
    },
    {
      header: 'Promo',
      key: 'promo',
      sortable: true,
      sortFn: (a, b) => {
        const valA = a.is_promo ? 1 : 0;
        const valB = b.is_promo ? 1 : 0;
        return valA - valB;
      },
      accessor: (row) => row.is_promo ? (
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
      )
    },
    { 
      header: 'Berangkat', 
      key: 'berangkat_tanggal',
      sortable: true,
      sortFn: (a, b) => {
        const tA = a.berangkat_tanggal ? new Date(a.berangkat_tanggal).getTime() : 0;
        const tB = b.berangkat_tanggal ? new Date(b.berangkat_tanggal).getTime() : 0;
        return tA - tB;
      },
      accessor: (row) => formatDate(row.berangkat_tanggal)
    },
    { 
      header: 'Sisa Seat', 
      key: 'seat_sisa',
      sortable: true,
      sortFn: (a, b) => (a.seat_sisa || 0) - (b.seat_sisa || 0),
      accessor: (row) => row.seat_sisa ?? '-'
    },
    { 
      header: 'Harga Mulai', 
      key: 'harga',
      sortable: true,
      sortFn: (a, b) => getMinPrice(a) - getMinPrice(b),
      accessor: (row) => {
        const minPrice = getMinPrice(row);
        return minPrice > 0 ? formatCurrency(minPrice) : '-';
      }
    }
  ];

  return (
    <div className="space-y-6">
      <PageHeader 
        title="Daftar Paket" 
      />

      <UrgentPackagesBanner 
        packages={urgentPackages} 
        onOpenDetail={handleOpenDetail} 
      />

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
            searchPlaceholder="Cari paket..."
            emptyMessage='Belum ada paket untuk travel Anda.'
            toolbarActions={
              <div className="flex flex-col sm:flex-row gap-2 w-full flex-wrap sm:flex-nowrap">
                {/* 1. Jadwal (Bulan) */}
                <CustomDropdown 
                  value={filterMonth}
                  onChange={(val) => setFilterMonth(val)}
                  className="!mb-0 w-full sm:w-40"
                  placeholder="Semua Jadwal"
                  options={[
                    { value: '', label: 'Semua Jadwal' },
                    ...availableMonths.map(m => ({ value: m.value, label: m.label }))
                  ]}
                />
                {/* 2. Maskapai */}
                <CustomDropdown 
                  value={filterMaskapai}
                  onChange={(val) => setFilterMaskapai(val)}
                  className="!mb-0 w-full sm:w-44"
                  placeholder="Semua Maskapai"
                  options={[
                    { value: '', label: 'Semua Maskapai' },
                    ...availableMaskapai.map(m => ({ value: m.value, label: m.label }))
                  ]}
                />
                {/* 3. Status */}
                <CustomDropdown 
                  value={filterStatus}
                  onChange={(val) => setFilterStatus(val)}
                  className="!mb-0 w-full sm:w-36"
                  placeholder="Semua Status"
                  options={[
                    { value: '', label: 'Semua Status' },
                    { value: 'draft', label: 'Draft' },
                    { value: 'published', label: 'Published' },
                    { value: 'archived', label: 'Archived' }
                  ]}
                />
              </div>
            }
          />
        </div>
      )}
    </div>
  );
};

export default PaketPage;
