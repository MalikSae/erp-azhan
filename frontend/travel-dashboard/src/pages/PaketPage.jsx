import React, { useState, useEffect, useMemo } from 'react';
import PageHeader from '../components/ui/PageHeader';
import Card from '../components/ui/Card';
import DataTable from '../components/ui/DataTable';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import Badge from '../components/ui/Badge';
import CustomDropdown from '../components/ui/CustomDropdown';
import { listSchedulesAdmin } from '../api/schedules';
import UrgentPackagesBanner from '../components/UrgentPackagesBanner';
import { Plane, Hotel, CheckCircle2, XCircle, Sparkles, Eye } from 'lucide-react';

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

const PaketPage = () => {
  const [schedules, setSchedules] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  
  const [filterStatus, setFilterStatus] = useState('');
  const [filterMonth, setFilterMonth] = useState('');

  // Modal Detail State (Read-Only)
  const [selectedSchedule, setSelectedSchedule] = useState(null);
  const [isDetailOpen, setIsDetailOpen] = useState(false);

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
    const schedule = pkgOrSchedule.rawSchedule || pkgOrSchedule;
    setSelectedSchedule(schedule);
    setIsDetailOpen(true);
  };

  const columns = [
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
            emptyMessage='Belum ada paket untuk travel Anda.'
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
                      onClick={() => handleOpenDetail(row)}
                      className="!px-2.5 text-primary-600 hover:text-primary-700 hover:bg-primary-50"
                      title="Lihat Detail Paket"
                    >
                      <Eye size={15} className="mr-1 inline" />
                      Detail
                    </Button>
                  </div>
                );
              }
              return row[key];
            }}
          />
        </div>
      )}

      {/* Modal Detail Paket (Read-Only) */}
      <Modal
        isOpen={isDetailOpen}
        onClose={() => setIsDetailOpen(false)}
        title="Detail Paket Umroh"
        size="lg"
      >
        {selectedSchedule && (
          <div className="space-y-6 font-body text-neutral-800">
            {/* Header Paket */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-4 border-b border-neutral-200">
              <div>
                <h3 className="text-lg font-bold font-heading text-neutral-900">
                  {selectedSchedule.jadwal_nama}
                </h3>
                <p className="text-xs text-neutral-500">
                  ID Jadwal: #{selectedSchedule.id}
                </p>
              </div>
              <div className="flex items-center gap-2">
                {selectedSchedule.is_promo && <Badge variant="promo">Promo</Badge>}
                {selectedSchedule.status && (
                  <Badge variant={selectedSchedule.status}>
                    {selectedSchedule.status.charAt(0).toUpperCase() + selectedSchedule.status.slice(1)}
                  </Badge>
                )}
              </div>
            </div>

            {/* Grid Detail Keberangkatan & Akomodasi */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* Kolom 1: Penerbangan & Jadwal */}
              <div className="p-4 bg-neutral-50 rounded-lg space-y-2 border border-neutral-200">
                <div className="flex items-center gap-2 text-primary-700 font-semibold font-heading text-xs uppercase tracking-wider pb-1 border-b border-neutral-200">
                  <Plane size={15} />
                  <span>Jadwal Penerbangan</span>
                </div>
                <div className="text-xs space-y-1.5 pt-1">
                  <p>
                    <span className="text-neutral-500">Maskapai:</span>{' '}
                    <span className="font-semibold text-neutral-900">{selectedSchedule.maskapai?.name || '-'}</span>
                  </p>
                  <p>
                    <span className="text-neutral-500">Keberangkatan:</span>{' '}
                    <span className="font-medium text-neutral-900">
                      {formatDate(selectedSchedule.berangkat_tanggal)} ({selectedSchedule.berangkat_jam || '-'})
                    </span>
                  </p>
                  {selectedSchedule.berangkat_kode_penerbangan && (
                    <p>
                      <span className="text-neutral-500">Kode Berangkat:</span>{' '}
                      <span className="font-mono text-neutral-900">{selectedSchedule.berangkat_kode_penerbangan}</span>
                    </p>
                  )}
                  {(selectedSchedule.berangkat_bandara_asal || selectedSchedule.berangkat_bandara_tujuan) && (
                    <p>
                      <span className="text-neutral-500">Rute Berangkat:</span>{' '}
                      <span className="font-medium text-neutral-900">{selectedSchedule.berangkat_bandara_asal || '-'} → {selectedSchedule.berangkat_bandara_tujuan || '-'}</span>
                    </p>
                  )}
                  <p>
                    <span className="text-neutral-500">Kepulangan:</span>{' '}
                    <span className="font-medium text-neutral-900">
                      {formatDate(selectedSchedule.pulang_tanggal)} ({selectedSchedule.pulang_jam || '-'})
                    </span>
                  </p>
                  {selectedSchedule.pulang_kode_penerbangan && (
                    <p>
                      <span className="text-neutral-500">Kode Pulang:</span>{' '}
                      <span className="font-mono text-neutral-900">{selectedSchedule.pulang_kode_penerbangan}</span>
                    </p>
                  )}
                  {(selectedSchedule.pulang_bandara_asal || selectedSchedule.pulang_bandara_tujuan) && (
                    <p>
                      <span className="text-neutral-500">Rute Pulang:</span>{' '}
                      <span className="font-medium text-neutral-900">{selectedSchedule.pulang_bandara_asal || '-'} → {selectedSchedule.pulang_bandara_tujuan || '-'}</span>
                    </p>
                  )}
                  <p>
                    <span className="text-neutral-500">Tipe Penerbangan:</span>{' '}
                    <span className="font-medium">{selectedSchedule.is_direct_flight ? 'Direct Flight (Langsung)' : 'Transit'}</span>
                  </p>
                  {!selectedSchedule.is_direct_flight && selectedSchedule.transit_bandara && (
                    <p><span className="text-neutral-500">Transit:</span>{' '}<span className="font-medium text-neutral-900">{selectedSchedule.transit_bandara}</span></p>
                  )}
                </div>
              </div>

              {/* Kolom 2: Hotel & Kursi */}
              <div className="p-4 bg-neutral-50 rounded-lg space-y-2 border border-neutral-200">
                <div className="flex items-center gap-2 text-primary-700 font-semibold font-heading text-xs uppercase tracking-wider pb-1 border-b border-neutral-200">
                  <Hotel size={15} />
                  <span>Akomodasi & Kuota</span>
                </div>
                <div className="text-xs space-y-1.5 pt-1">
                  <p>
                    <span className="text-neutral-500">Hotel Makkah:</span>{' '}
                    <span className="font-semibold text-neutral-900">
                      {selectedSchedule.hotel_mekkah?.name || '-'} {selectedSchedule.hotel_mekkah?.star_rating ? `(★${selectedSchedule.hotel_mekkah.star_rating})` : ''}
                    </span>
                  </p>
                  <p>
                    <span className="text-neutral-500">Hotel Madinah:</span>{' '}
                    <span className="font-semibold text-neutral-900">
                      {selectedSchedule.hotel_madinah?.name || '-'} {selectedSchedule.hotel_madinah?.star_rating ? `(★${selectedSchedule.hotel_madinah.star_rating})` : ''}
                    </span>
                  </p>
                  <p>
                    <span className="text-neutral-500">Total Kuota:</span>{' '}
                    <span className="font-medium text-neutral-900">{selectedSchedule.seat_total} Kursi</span>
                  </p>
                  <p>
                    <span className="text-neutral-500">Sisa Kuota:</span>{' '}
                    <span className="font-bold text-success-600">{selectedSchedule.seat_sisa} Kursi Tersedia</span>
                  </p>
                </div>
              </div>
            </div>

            {/* Rincian Harga Kamar */}
            <div>
              <h4 className="text-xs font-bold font-heading uppercase tracking-wider text-neutral-500 mb-2">
                Rincian Harga Paket per Tipe Kamar
              </h4>
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                <div className="p-3 bg-white border border-neutral-200 rounded-lg text-center shadow-sm">
                  <span className="text-xs font-semibold uppercase text-neutral-500 block">Kamar Quad (4 Orang)</span>
                  <span className="text-base font-bold font-heading text-primary-700">
                    {formatCurrency(selectedSchedule.harga_quad)}
                  </span>
                </div>
                <div className="p-3 bg-white border border-neutral-200 rounded-lg text-center shadow-sm">
                  <span className="text-xs font-semibold uppercase text-neutral-500 block">Kamar Triple (3 Orang)</span>
                  <span className="text-base font-bold font-heading text-primary-700">
                    {formatCurrency(selectedSchedule.harga_triple)}
                  </span>
                </div>
                <div className="p-3 bg-white border border-neutral-200 rounded-lg text-center shadow-sm">
                  <span className="text-xs font-semibold uppercase text-neutral-500 block">Kamar Double (2 Orang)</span>
                  <span className="text-base font-bold font-heading text-primary-700">
                    {formatCurrency(selectedSchedule.harga_double)}
                  </span>
                </div>
              </div>
            </div>

            {/* Fasilitas Include & Exclude */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* Include */}
              <div className="p-3.5 bg-white border border-neutral-200 rounded-lg space-y-2">
                <div className="flex items-center gap-1.5 text-xs font-bold font-heading uppercase tracking-wider text-success-700">
                  <CheckCircle2 size={15} />
                  <span>Fasilitas Termasuk (Include)</span>
                </div>
                {selectedSchedule.include_items && selectedSchedule.include_items.length > 0 ? (
                  <ul className="text-xs space-y-1 text-neutral-600 list-disc list-inside">
                    {selectedSchedule.include_items.map((inc, i) => (
                      <li key={i}>{inc}</li>
                    ))}
                  </ul>
                ) : (
                  <p className="text-xs text-neutral-400 italic">Tidak ada keterangan fasilitas include.</p>
                )}
              </div>

              {/* Exclude */}
              <div className="p-3.5 bg-white border border-neutral-200 rounded-lg space-y-2">
                <div className="flex items-center gap-1.5 text-xs font-bold font-heading uppercase tracking-wider text-neutral-500">
                  <XCircle size={15} />
                  <span>Tidak Termasuk (Exclude)</span>
                </div>
                {selectedSchedule.exclude_items && selectedSchedule.exclude_items.length > 0 ? (
                  <ul className="text-xs space-y-1 text-neutral-600 list-disc list-inside">
                    {selectedSchedule.exclude_items.map((exc, i) => (
                      <li key={i}>{exc}</li>
                    ))}
                  </ul>
                ) : (
                  <p className="text-xs text-neutral-400 italic">Tidak ada keterangan fasilitas exclude.</p>
                )}
              </div>
            </div>

            {/* Add-ons yang Tersedia */}
            {selectedSchedule.add_ons && selectedSchedule.add_ons.length > 0 && (
              <div>
                <h4 className="text-xs font-bold font-heading uppercase tracking-wider text-neutral-500 mb-2">
                  Layanan Tambahan (Add-Ons) Tersedia
                </h4>
                <div className="flex flex-wrap gap-2">
                  {selectedSchedule.add_ons.map((addon) => (
                    <span
                      key={addon.id}
                      className="inline-flex items-center gap-1 px-2.5 py-1 bg-primary-50 text-primary-800 text-xs font-medium rounded-md border border-primary-200"
                    >
                      <Sparkles size={12} className="text-primary-600" />
                      {addon.name}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
};

export default PaketPage;
