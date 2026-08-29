import React, { useState, useEffect, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import PageHeader from '../components/ui/PageHeader';
import DataTable from '../components/ui/DataTable';
import Button from '../components/ui/Button';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import Badge from '../components/ui/Badge';
import CustomDropdown from '../components/ui/CustomDropdown';
import { listSchedulesAdmin } from 'shared';
import UrgentPackagesBanner from '../components/UrgentPackagesBanner';
import { Eye, Files, Check, Archive, Minus, Zap } from 'lucide-react';

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

  const counts = useMemo(() => {
    return {
      all: schedules.length,
      urgent: schedules.filter(isScheduleUrgent).length,
      published: schedules.filter(s => s.status === 'published').length,
      draft: schedules.filter(s => s.status === 'draft').length,
      archived: schedules.filter(s => s.status === 'archived').length,
    };
  }, [schedules, today]);

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

      // 2. Maskapai Filter
      if (filterMaskapai) {
        if (!s.maskapai || String(s.maskapai.id) !== String(filterMaskapai)) return false;
      }

      // 3. Month Filter
      if (filterMonth) {
        if (!s.berangkat_tanggal) return false;
        const d = new Date(s.berangkat_tanggal);
        if (isNaN(d.getTime())) return false;
        const val = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
        if (val !== filterMonth) return false;
      }

      return true;
    });
  }, [schedules, activeTab, filterMaskapai, filterMonth, today]);

  const tabs = [
    { id: 'all', label: 'Semua Paket', count: counts.all },
    { id: 'urgent', label: '🚨 Mendesak (<60 Hari)', count: counts.urgent, isUrgent: true },
    { id: 'published', label: 'Published', count: counts.published },
    { id: 'draft', label: 'Draft', count: counts.draft },
    { id: 'archived', label: 'Archived', count: counts.archived },
  ];

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
      header: 'Publish', 
      key: 'status',
      align: 'center',
      accessor: (row) => {
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
    },
    { 
      header: 'Tiket', 
      key: 'tiket',
      align: 'center',
      sortable: true,
      sortFn: (a, b) => {
        const valA = a.is_ticket_confirmed ? 1 : 0;
        const valB = b.is_ticket_confirmed ? 1 : 0;
        return valA - valB;
      },
      accessor: (row) => (
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
      )
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
      },
      accessor: (row) => (
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
      accessor: (row) => {
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
    },
    { 
      header: 'Sisa Seat', 
      key: 'seat_sisa',
      sortable: true,
      sortFn: (a, b) => (a.seat_sisa || 0) - (b.seat_sisa || 0),
      accessor: (row) => {
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

      {errorMessage && (
        <Alert variant="error">{errorMessage}</Alert>
      )}

      {/* Quick Filter Tabs */}
      <div className="flex flex-wrap items-center gap-2 border-b border-neutral-200 pb-3">
        {tabs.map((tab) => {
          const isActive = activeTab === tab.id;
          const isUrgentTab = tab.id === 'urgent';
          const hasUrgentCount = isUrgentTab && tab.count > 0;

          let btnClass = 'bg-white text-neutral-600 hover:bg-neutral-50 border border-neutral-200/90';
          let badgeClass = 'bg-neutral-100 text-neutral-600';

          if (isActive) {
            btnClass = isUrgentTab ? 'bg-danger-600 text-white shadow-2xs font-bold' : 'bg-[#181C1F] text-white shadow-2xs font-bold';
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
            searchPlaceholder="Cari paket..."
            emptyMessage={activeTab === 'urgent' ? 'Tidak ada paket mendesak saat ini.' : 'Belum ada paket untuk travel Anda.'}
            toolbarActions={
              <div className="flex flex-wrap items-center gap-2">
                {/* 1. Jadwal (Bulan) */}
                <CustomDropdown 
                  value={filterMonth}
                  onChange={(val) => setFilterMonth(val)}
                  className="!mb-0 w-40"
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
                  className="!mb-0 w-44"
                  placeholder="Semua Maskapai"
                  options={[
                    { value: '', label: 'Semua Maskapai' },
                    ...availableMaskapai.map(m => ({ value: m.value, label: m.label }))
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
