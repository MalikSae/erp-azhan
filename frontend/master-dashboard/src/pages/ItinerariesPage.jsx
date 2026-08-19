import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import PageHeader from '../components/ui/PageHeader';
import DataTable from '../components/ui/DataTable';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import { listItineraries, deleteItinerary } from '../api/itineraries';

const ItinerariesPage = () => {
  const navigate = useNavigate();
  const [itineraries, setItineraries] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  
  const [deleteConfirmId, setDeleteConfirmId] = useState(null);

  const fetchItineraries = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const data = await listItineraries();
      setItineraries(data);
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

  return (
    <div className="space-y-6">
      <PageHeader 
        title="Kelola Itinerary" 
        actionLabel="+ Tambah Itinerary" 
        onAction={() => navigate('/itineraries/new')}
      />

      {errorMessage && (
        <Alert variant="error">{errorMessage}</Alert>
      )}

      {isLoading ? (
        <div className="flex justify-center p-8 bg-white rounded-lg border border-neutral-200 shadow-sm">
          <LoadingSpinner />
        </div>
      ) : (
        <DataTable
          columns={[
            { header: 'Judul', key: 'title' },
            { header: 'Jumlah Hari', key: 'day_count' },
            { header: 'Aksi', key: 'aksi' },
          ]}
          data={itineraries}
          itemsPerPage={10}
          emptyMessage='Belum ada itinerary. Klik "+ Tambah Itinerary" untuk menambahkan.'
          renderCell={(row, key) => {
            if (key === 'aksi') return (
              <div className="flex gap-3">
                <button 
                  onClick={() => navigate(`/itineraries/${row.id}/edit`)} 
                  title="Edit"
                  className="text-neutral-400 hover:text-neutral-700 transition-colors"
                >
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                  </svg>
                </button>
                <button 
                  onClick={() => confirmDelete(row.id)} 
                  title="Hapus"
                  className="text-neutral-400 hover:text-neutral-700 transition-colors"
                >
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              </div>
            );
            return row[key];
          }}
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
