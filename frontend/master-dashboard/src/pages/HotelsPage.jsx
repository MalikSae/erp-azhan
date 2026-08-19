import React, { useState, useEffect } from 'react';
import PageHeader from '../components/ui/PageHeader';
import Card from '../components/ui/Card';
import DataTable from '../components/ui/DataTable';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import FormField from '../components/ui/FormField';
import Input from '../components/ui/Input';
import Select from '../components/ui/Select';
import CustomDropdown from '../components/ui/CustomDropdown';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import { listHotels, createHotel, updateHotel, deleteHotel } from '../api/hotels';
import { uploadMedia } from '../api/media';

const initialForm = {
  name: '',
  city: '',
  star_rating: '',
  distance_m: '',
  photo_url: ''
};

const PhotoCell = ({ url, name }) => {
  const [error, setError] = React.useState(false);
  
  const placeholder = (
    <div className="w-8 h-8 bg-neutral-100 rounded border border-neutral-200 flex items-center justify-center text-neutral-400 text-xs font-medium">
      —
    </div>
  );
  
  if (!url || error) return placeholder;
  
  return (
    <img 
      src={url.startsWith('/') ? `${import.meta.env.VITE_API_BASE_URL}${url}` : url} 
      alt={`${name} foto`}
      className="w-8 h-8 object-contain rounded bg-white"
      onError={() => setError(true)}
    />
  );
};

const HotelsPage = () => {
  const [hotels, setHotels] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  
  const [cityFilter, setCityFilter] = useState('');
  
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingHotel, setEditingHotel] = useState(null);
  const [formData, setFormData] = useState(initialForm);
  const [formErrors, setFormErrors] = useState(null);
  const [isUploading, setIsUploading] = useState(false);
  const [localPreview, setLocalPreview] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  const [deleteConfirmId, setDeleteConfirmId] = useState(null);

  const fetchHotels = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const data = await listHotels();
      setHotels(data);
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal memuat data hotel.";
      setErrorMessage(msg);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchHotels();
  }, []);

  const handleOpenModal = (hotel = null) => {
    if (hotel) {
      setEditingHotel(hotel);
      
      const rawCity = (hotel.city || '').toLowerCase();
      let normalizedCity = '';
      if (rawCity === 'makkah' || rawCity === 'mekkah') normalizedCity = 'Makkah';
      else if (rawCity === 'madinah') normalizedCity = 'Madinah';
      else normalizedCity = hotel.city || '';
      
      setFormData({
        name: hotel.name,
        city: normalizedCity,
        star_rating: hotel.star_rating !== null ? hotel.star_rating.toString() : '',
        distance_m: hotel.distance_m === null ? '' : hotel.distance_m.toString(),
        photo_url: hotel.photo_url || ''
      });
      setLocalPreview(null);
    } else {
      setEditingHotel(null);
      setFormData(initialForm);
      setLocalPreview(null);
    }
    setFormErrors(null);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    if (!isSubmitting) {
      setIsModalOpen(false);
      setEditingHotel(null);
      setFormData(initialForm);
      setLocalPreview(null);
    }
  };

  const handleFileChange = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    // Local preview
    const objectUrl = URL.createObjectURL(file);
    setLocalPreview(objectUrl);
    setFormErrors(null);
    setIsUploading(true);

    try {
      const url = await uploadMedia(file, 'hotel-photos');
      setFormData(prev => ({ ...prev, photo_url: url }));
    } catch (err) {
      console.error(err);
      setFormErrors('Gagal upload foto, coba lagi');
    } finally {
      setIsUploading(false);
    }
  };

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setFormErrors(null);
    setIsSubmitting(true);

    try {
      const payload = {
        name: formData.name,
        city: formData.city,
        star_rating: parseInt(formData.star_rating, 10),
        distance_m: formData.distance_m === '' ? null : parseInt(formData.distance_m, 10),
        photo_url: formData.photo_url === '' ? null : formData.photo_url
      };

      if (editingHotel) {
        await updateHotel(editingHotel.id, payload);
      } else {
        await createHotel(payload);
      }

      handleCloseModal();
      fetchHotels();
    } catch (error) {
      const msg = error.response?.data?.error || "Terjadi kesalahan, coba lagi.";
      setFormErrors(msg);
    } finally {
      setIsSubmitting(false);
    }
  };

  const confirmDelete = (id) => {
    setDeleteConfirmId(id);
  };

  const cancelDelete = () => {
    setDeleteConfirmId(null);
  };

  const handleDelete = async () => {
    try {
      await deleteHotel(deleteConfirmId);
      setDeleteConfirmId(null);
      fetchHotels();
    } catch (error) {
      setDeleteConfirmId(null);
      if (error.response?.status === 409) {
        setErrorMessage(error.response?.data?.error || "Tidak bisa dihapus, masih dipakai oleh paket lain.");
      } else {
        setErrorMessage("Terjadi kesalahan saat menghapus data.");
      }
    }
  };

  const hotelToDelete = hotels.find(h => h.id === deleteConfirmId);

  const filteredHotels = hotels.filter(hotel => {
    if (cityFilter) {
      const hCity = (hotel.city || '').toLowerCase().replace('mekkah', 'makkah');
      const fCity = cityFilter.toLowerCase().replace('mekkah', 'makkah');
      if (hCity !== fCity) {
        return false;
      }
    }
    return true;
  });

  return (
    <div className="space-y-6">
      <PageHeader 
        title="Kelola Hotel" 
        actionLabel="+ Tambah Hotel" 
        onAction={() => handleOpenModal()}
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
            { header: 'Foto', key: 'photo_url' },
            { header: 'Nama', key: 'name' },
            { header: 'Kota', key: 'city' },
            { header: 'Bintang', key: 'star_rating' },
            { header: 'Jarak (m)', key: 'distance_m' },
            { header: 'Aksi', key: 'aksi' },
          ]}
          data={filteredHotels}
          toolbarActions={
            <CustomDropdown
              value={cityFilter}
              onChange={(val) => setCityFilter(val)}
              options={[
                { value: '', label: 'Semua Kota' },
                { value: 'Makkah', label: 'Makkah' },
                { value: 'Madinah', label: 'Madinah' }
              ]}
              placeholder="Pilih Kota"
              className="!mb-0 min-w-[180px]"
            />
          }
          itemsPerPage={10}
          emptyMessage='Belum ada hotel. Klik "+ Tambah Hotel" untuk menambahkan.'
          renderCell={(row, key) => {
            if (key === 'photo_url') {
              return <PhotoCell url={row.photo_url} name={row.name} />;
            }
            if (key === 'star_rating') return `${row.star_rating} bintang`;
            if (key === 'distance_m') return row.distance_m !== null ? `±${row.distance_m}m` : '-';
            if (key === 'aksi') return (
              <div className="flex gap-3">
                <button 
                  onClick={() => handleOpenModal(row)} 
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

      {/* Form Modal */}
      <Modal 
        isOpen={isModalOpen} 
        onClose={handleCloseModal}
        title={editingHotel ? "Edit Hotel" : "Tambah Hotel"}
        size="md"
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          <FormField label="Nama Hotel" required>
            <Input 
              name="name"
              value={formData.name}
              onChange={handleChange}
              required
              placeholder="Contoh: Hilton Makkah"
            />
          </FormField>
          
          <CustomDropdown
            label="Kota"
            required
            value={formData.city}
            onChange={(val) => handleChange({ target: { name: 'city', value: val }})}
            options={[
              { value: 'Makkah', label: 'Makkah' },
              { value: 'Madinah', label: 'Madinah' }
            ]}
            placeholder="Pilih Kota"
          />
          
          <CustomDropdown
            label="Rating Bintang"
            required
            value={formData.star_rating}
            onChange={(val) => handleChange({ target: { name: 'star_rating', value: val }})}
            options={[
              { value: '1', label: '1 Bintang' },
              { value: '2', label: '2 Bintang' },
              { value: '3', label: '3 Bintang' },
              { value: '4', label: '4 Bintang' },
              { value: '5', label: '5 Bintang' }
            ]}
            placeholder="Pilih Rating Bintang"
          />
          
          <FormField label="Jarak ke Masjid (meter)">
            <Input 
              type="number"
              name="distance_m"
              value={formData.distance_m}
              onChange={handleChange}
              placeholder="Contoh: 200 (opsional)"
            />
          </FormField>

          <FormField label="Foto Hotel (Opsional)">
            {(localPreview || formData.photo_url) && (
              <div className="mb-2 relative inline-block">
                <img 
                  src={localPreview || (formData.photo_url.startsWith('/') ? `${import.meta.env.VITE_API_BASE_URL}${formData.photo_url}` : formData.photo_url)}
                  alt="Preview"
                  className="h-16 object-contain rounded bg-white border border-neutral-200"
                />
                {(localPreview || formData.photo_url) && (
                  <button
                    type="button"
                    onClick={() => {
                      setLocalPreview(null);
                      setFormData(prev => ({ ...prev, photo_url: '' }));
                    }}
                    className="absolute -top-2 -right-2 bg-danger-600 text-white rounded-full w-5 h-5 flex items-center justify-center shadow-sm text-[10px]"
                  >
                    X
                  </button>
                )}
              </div>
            )}
            
            <Input 
              type="file"
              accept="image/*"
              onChange={handleFileChange}
              disabled={isUploading}
            />
            {isUploading && <p className="text-sm text-neutral-500 mt-1">Mengupload...</p>}
          </FormField>

          {formErrors && (
            <Alert variant="error">{formErrors}</Alert>
          )}

          <div className="pt-4 flex justify-end space-x-3 border-t border-neutral-200">
            <Button type="button" variant="ghost" onClick={handleCloseModal} disabled={isSubmitting}>
              Batal
            </Button>
            <Button type="submit" variant="primary" isLoading={isSubmitting}>
              {editingHotel ? "Update" : "Simpan"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteConfirmId !== null}
        onClose={cancelDelete}
        title="Hapus Hotel?"
        size="sm"
        footer={
          <>
            <Button variant="ghost" onClick={cancelDelete}>Batal</Button>
            <Button variant="danger" onClick={handleDelete}>Hapus</Button>
          </>
        }
      >
        <p className="text-neutral-600 font-body">
          Yakin ingin menghapus {hotelToDelete?.name}? Tindakan ini tidak bisa dibatalkan.
        </p>
      </Modal>
    </div>
  );
};

export default HotelsPage;
