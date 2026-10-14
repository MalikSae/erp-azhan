import React, { useState, useEffect, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import PageHeader from 'shared/ui/PageHeader';
import DataTable from 'shared/ui/DataTable';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import StatTile from 'shared/ui/StatTile';
import CustomDropdown from 'shared/ui/CustomDropdown';
import ActionMenu from 'shared/ui/ActionMenu';
import { listSchedulesAdmin } from 'shared';
import { Eye, Package, CheckCircle2, FileText, Clock3, CalendarDays } from 'lucide-react';

const STATUS_STYLE = {
  published: { label: 'Published', chip: 'bg-success-50 text-success-700 border-success-200', dot: 'bg-success-500' },
  draft: { label: 'Draft', chip: 'bg-warning-50 text-warning-700 border-warning-200', dot: 'bg-warning-500' },
  archived: { label: 'Archived', chip: 'bg-neutral-50 text-neutral-600 border-neutral-200', dot: 'bg-neutral-400' },
};

const StatusPill = ({ status }) => {
  const s = STATUS_STYLE[status];
  if (!s) return <span className="text-xs text-neutral-400 font-body">-</span>;
  return (
    <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-medium font-body border ${s.chip}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${s.dot}`} />
      {s.label}
    </span>
  );
};

// Helper: hitung hari tersisa sampai keberangkatan
const getDaysRemaining = (dateStr) => {
  if (!dateStr) return null;
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return null;
  const now = new Date();
  now.setHours(0, 0, 0, 0);
  d.setHours(0, 0, 0, 0);
  return Math.ceil((d - now) / 86400000);
};

// Helper: apakah paket tergolong mendesak (< 60 hari & belum full)
const isScheduleUrgent = (schedule) => {
  if (schedule.status !== 'published') return false;
  const days = getDaysRemaining(schedule.berangkat_tanggal);
  if (days === null || days < 0 || days > 60) return false;
  const seatSisa = schedule.seat_sisa ?? 0;
  return seatSisa > 0;
};

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
  const [activeTab, setActiveTab] = useState('all');

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
  }, [schedules]);

  const filteredSchedules = useMemo(() => {
    return schedules.map(s => ({
      ...s,
      // Kolom bantu pencarian: maskapai ikut bisa dicari.
      _cari: s.maskapai?.name || '',
    })).filter(s => {
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
  }, [schedules, activeTab, filterMaskapai, filterMonth]);

  const tabs = [
    { id: 'all', label: 'Semua Paket', count: counts.all },
    { id: 'urgent', label: 'Mendesak (<60 Hari)', count: counts.urgent },
    { id: 'published', label: 'Published', count: counts.published },
    { id: 'draft', label: 'Draft', count: counts.draft },
    { id: 'archived', label: 'Archived', count: counts.archived },
  ];

  const columns = [
    { 
      header: 'Nama Paket', 
      key: 'jadwal_nama',
      accessor: (row) => {
        const days = getDaysRemaining(row.berangkat_tanggal);
        const isUrgent = isScheduleUrgent(row);
        return (
          <div className="flex flex-col items-start gap-0.5">
            <button
              type="button"
              onClick={(e) => { e.stopPropagation(); navigate(`/schedules/${row.id}`); }}
              className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors cursor-pointer"
              title="Lihat Detail Paket"
            >
              {row.jadwal_nama}
            </button>
            <div className="flex items-center gap-1.5 text-[11px] font-body text-neutral-400">
              <CalendarDays className="w-3.5 h-3.5" strokeWidth={1.75} />
              <span>{formatDate(row.berangkat_tanggal)}</span>
              {isUrgent && days !== null && (
                <span
                  className="text-danger-600 font-medium"
                  title="Mendekati keberangkatan & belum penuh"
                >
                  · {days} hari lagi
                </span>
              )}
            </div>
          </div>
        );
      }
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
      accessor: (row) => <StatusPill status={row.status} />
    },
    {
      header: 'Sisa Seat', 
      key: 'seat_sisa',
      sortable: true,
      sortFn: (a, b) => (a.seat_sisa || 0) - (b.seat_sisa || 0),
      accessor: (row) => {
        const sisa = row.seat_sisa !== null && row.seat_sisa !== undefined ? Number(row.seat_sisa) : null;
        if (sisa !== null && sisa <= 0) {
          return (
            <span className="inline-flex items-center px-2 py-0.5 rounded-md text-[11px] font-bold font-heading tracking-wide uppercase bg-neutral-900 text-white">
              Penuh
            </span>
          );
        }
        const isUrgent = isScheduleUrgent(row);
        const total = Number(row.seat_total) || 0;
        const filled = Math.max(0, total - (row.seat_sisa || 0));
        const percent = total > 0 ? Math.min(100, (filled / total) * 100) : 0;
        const barColor = isUrgent ? 'bg-danger-500' : percent >= 80 ? 'bg-warning-500' : 'bg-primary-500';
        return (
          <div className="min-w-[110px]">
            <div className="flex items-baseline gap-1 font-body">
              <span className={`text-sm font-semibold ${isUrgent ? 'text-danger-700' : 'text-neutral-900'}`}>
                {row.seat_sisa ?? '-'}
              </span>
              <span className="text-[11px] text-neutral-400">sisa dari {row.seat_total ?? '-'} pax</span>
            </div>
            {row.seat_total > 0 && (
              <div className="mt-1 h-1.5 w-full bg-neutral-100 rounded-full overflow-hidden" title={`Terisi ${filled}/${row.seat_total} pax`}>
                <div className={`h-full rounded-full ${barColor}`} style={{ width: `${percent}%` }} />
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
        if (minPrice <= 0) return <span className="text-xs text-neutral-400 font-body">-</span>;
        return <span className="text-[13px] font-semibold text-neutral-900 font-body whitespace-nowrap">{formatCurrency(minPrice)}</span>;
      }
    },
    {
      header: 'Aksi',
      key: 'aksi',
      align: 'center',
      accessor: (row) => {
        const menuItems = [
          {
            label: 'Lihat',
            icon: Eye,
            onClick: () => navigate(`/schedules/${row.id}`),
            tooltip: 'Lihat Detail Paket'
          }
        ];
        return (
          <div className="flex items-center justify-center">
            <ActionMenu items={menuItems} align="right" />
          </div>
        );
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

      {!isLoading && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatTile icon={Package} variant="primary" value={counts.all} label="Total Paket" />
          <StatTile icon={CheckCircle2} variant="success" value={counts.published} label="Published" />
          <StatTile icon={FileText} variant="warning" value={counts.draft} label="Draft" />
          <StatTile icon={Clock3} variant="danger" value={counts.urgent} label="Mendesak" hint="Berangkat < 60 hari" />
        </div>
      )}

      {isLoading ? (
        <div className="flex justify-center p-8 bg-white rounded-2xl border border-neutral-200 shadow-card">
          <LoadingSpinner />
        </div>
      ) : (
        <div className="space-y-4">
          <DataTable
            columns={columns}
            data={filteredSchedules}
            itemsPerPage={15}
            onRowClick={(row) => navigate(`/schedules/${row.id}`)}
            searchPlaceholder="Cari nama paket atau maskapai..."
            emptyMessage={activeTab === 'urgent' ? 'Tidak ada paket mendesak saat ini.' : 'Belum ada paket untuk travel Anda.'}
            toolbarActions={
              <div className="flex flex-wrap items-center gap-2">
                {/* 0. Status */}
                <CustomDropdown
                  value={activeTab}
                  onChange={(val) => setActiveTab(val)}
                  className="!mb-0 w-52"
                  placeholder="Semua Status"
                  options={tabs.map(tab => ({ value: tab.id, label: `${tab.label} (${tab.count})` }))}
                />
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
