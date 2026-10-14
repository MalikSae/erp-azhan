import React, { useState, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { CalendarDays, Pencil, Trash2 } from 'lucide-react';
import PageHeader from 'shared/ui/PageHeader';
import DataTable from 'shared/ui/DataTable';
import Button from 'shared/ui/Button';
import Badge from 'shared/ui/Badge';
import Modal from 'shared/ui/Modal';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import ActionMenu from 'shared/ui/ActionMenu';
import { listItineraries, deleteItinerary } from '../api/itineraries';

const formatDate = (dateStr) => {
  if (!dateStr) return '-';
  const d = new Date(dateStr);
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
  return `${d.getDate()} ${months[d.getMonth()]} ${d.getFullYear()}`;
};

const ItinerariesPage = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [successMessage, setSuccessMessage] = useState(null);
  const [itineraries, setItineraries] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);

  const [deleteConfirmId, setDeleteConfirmId] = useState(null);

  const fetchItineraries = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const data = await listItineraries();
      setItineraries(data || []);
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal memuat data itinerary.";
      setErrorMessage(msg);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchItineraries();
  }, []);

  // Pesan sukses dari form (simpan/ubah); dibersihkan agar tidak muncul lagi saat refresh.
  useEffect(() => {
    if (location.state?.successMessage) {
      setSuccessMessage(location.state.successMessage);
      window.history.replaceState({}, '');
    }
  }, [location.state]);

  const confirmDelete = (id) => {
    setDeleteConfirmId(id);
  };

  const cancelDelete = () => {
    setDeleteConfirmId(null);
  };

  const handleDelete = async () => {
    try {
      await deleteItinerary(deleteConfirmId);
      setDeleteConfirmId(null);
      fetchItineraries();
    } catch (error) {
      setDeleteConfirmId(null);
      if (error.response?.status === 409) {
        setErrorMessage(error.response?.data?.error || "Tidak bisa dihapus, masih dipakai oleh entitas lain.");
      } else {
        setErrorMessage("Terjadi kesalahan saat menghapus data.");
      }
    }
  };

  const itineraryToDelete = itineraries.find(i => i.id === deleteConfirmId);

  const columns = [
    { header: 'Itinerary', key: 'title', sortable: true },
    { header: 'Kategori', key: 'category_name', sortable: true },
    { header: 'Durasi', key: 'day_count', sortable: true },
    { header: 'Dibuat', key: 'created_at', sortable: true },
    { header: 'Aksi', key: 'aksi' },
  ];

  const renderCell = (row, key) => {
    if (key === 'title') {
      return (
        <button
          type="button"
          onClick={() => navigate(`/itineraries/${row.id}/edit`)}
          className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors"
        >
          {row.title}
        </button>
      );
    }

    if (key === 'category_name') {
      return row.category_name
        ? <Badge variant="neutral" hideIcon className="!bg-white !border-neutral-300 !text-neutral-900">{row.category_name}</Badge>
        : <span className="text-xs text-neutral-400 font-body">Semua kategori</span>;
    }

    if (key === 'day_count') {
      return <span className="text-[13px] font-body text-neutral-700 whitespace-nowrap">{row.day_count || 0} hari</span>;
    }

    if (key === 'created_at') {
      return (
        <span className="inline-flex items-center gap-1.5 text-[13px] font-body text-neutral-500 whitespace-nowrap">
          <CalendarDays className="w-3.5 h-3.5 text-neutral-400" strokeWidth={1.75} />
          {formatDate(row.created_at)}
        </span>
      );
    }

    if (key === 'aksi') {
      const menuItems = [
        { label: 'Edit', icon: Pencil, tooltip: 'Edit Itinerary', onClick: () => navigate(`/itineraries/${row.id}/edit`) },
        { label: 'Hapus', icon: Trash2, danger: true, tooltip: 'Hapus Itinerary', onClick: () => confirmDelete(row.id) },
      ];
      return <ActionMenu items={menuItems} align="right" />;
    }

    return row[key];
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Kelola Itinerary"
        actionLabel="+ Tambah Itinerary"
        onAction={() => navigate('/itineraries/new')}
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
          data={itineraries}
          itemsPerPage={10}
          emptyMessage='Belum ada itinerary. Klik "+ Tambah Itinerary" untuk menambahkan.'
          searchPlaceholder="Cari judul itinerary..."
          renderCell={renderCell}
        />
      )}

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteConfirmId !== null}
        onClose={cancelDelete}
        title="Hapus Itinerary?"
        size="md"
        footer={
          <>
            <Button variant="ghost" onClick={cancelDelete}>Batal</Button>
            <Button variant="danger" onClick={handleDelete}>Hapus</Button>
          </>
        }
      >
        <p className="text-neutral-600 font-body">
          Yakin ingin menghapus "{itineraryToDelete?.title}"? Jika itinerary ini masih dipakai oleh paket manapun, penghapusan akan ditolak sistem.
        </p>
      </Modal>
    </div>
  );
};

export default ItinerariesPage;
