import React, { useState, useEffect, useMemo } from 'react';
import { Hotel, MapPin, Star, Play, Pencil, Trash2 } from 'lucide-react';
import PageHeader from 'shared/ui/PageHeader';
import DataTable from 'shared/ui/DataTable';
import Button from 'shared/ui/Button';
import Modal from 'shared/ui/Modal';
import Input from 'shared/ui/Input';
import FileInput from 'shared/ui/FileInput';
import CustomDropdown from 'shared/ui/CustomDropdown';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import ActionMenu from 'shared/ui/ActionMenu';
import { listHotels, listHotelCities, createHotel, updateHotel, deleteHotel } from '../api/hotels';
import { uploadMedia } from '../api/media';

const initialForm = {
  name: '',
  city: '',
  star_rating: '',
  distance_m: '',
  photo_url: '',
  video_url: ''
};

// Ejaan "Mekkah" dan "Makkah" dihitung satu kota.
const cityKey = (city) => {
  const k = (city || '').trim().toLowerCase();
  return k === 'mekkah' ? 'makkah' : k;
};

const PhotoCell = ({ url, name }) => {
  const [error, setError] = React.useState(false);

  if (!url || error) {
    return (
      <div className="w-10 h-10 shrink-0 bg-neutral-100 rounded-lg border border-neutral-200 flex items-center justify-center text-neutral-400">
        <Hotel className="w-4 h-4" strokeWidth={1.75} />
      </div>
    );
  }

  return (
    <img
      src={url.startsWith('/') ? `${import.meta.env.VITE_API_BASE_URL}${url}` : url}
      alt={`${name} foto`}
      className="w-10 h-10 shrink-0 object-cover rounded-lg border border-neutral-200 bg-white"
      onError={() => setError(true)}
    />
  );
};

const HotelsPage = () => {
  const [hotels, setHotels] = useState([]);
  const [cities, setCities] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);

  const [cityFilter, setCityFilter] = useState('');

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingHotel, setEditingHotel] = useState(null);
  const [formData, setFormData] = useState(initialForm);
  const [isNewCity, setIsNewCity] = useState(false);
  const [formErrors, setFormErrors] = useState(null);
  const [isUploading, setIsUploading] = useState(false);
  const [localPreview, setLocalPreview] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const [deleteConfirmId, setDeleteConfirmId] = useState(null);

  const fetchHotelsAndCities = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const [hotelsData, citiesData] = await Promise.all([
        listHotels(),
        listHotelCities()
      ]);
      setHotels(hotelsData || []);
      setCities(citiesData || []);
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal memuat data hotel.";
      setErrorMessage(msg);
    } finally {
      setIsLoading(false);
    }
  };

  const fetchCities = async () => {
    try {
      const data = await listHotelCities();
      setCities(data || []);
      return data || [];
    } catch (err) {
      console.error("Gagal memuat daftar kota:", err);
      return [];
    }
  };

  useEffect(() => {
    fetchHotelsAndCities();
  }, []);

  const handleOpenModal = async (hotel = null) => {
    const currentCities = await fetchCities();

    if (hotel) {
      setEditingHotel(hotel);
      const cityExists = currentCities.some(c => c.toLowerCase() === (hotel.city || '').toLowerCase());
      if (hotel.city && !cityExists) {
        setIsNewCity(true);
      } else {
        setIsNewCity(false);
      }

      setFormData({
        name: hotel.name || '',
        city: hotel.city || '',
        star_rating: hotel.star_rating !== null ? hotel.star_rating.toString() : '',
        distance_m: hotel.distance_m === null ? '' : hotel.distance_m.toString(),
        photo_url: hotel.photo_url || '',
        video_url: hotel.video_url || ''
      });
      setLocalPreview(null);
    } else {
      setEditingHotel(null);
      setIsNewCity(false);
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
      setIsNewCity(false);
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
        name: formData.name.trim(),
        city: formData.city.trim(),
        star_rating: parseInt(formData.star_rating, 10),
        distance_m: formData.distance_m === '' ? null : parseInt(formData.distance_m, 10),
        photo_url: formData.photo_url === '' ? null : formData.photo_url,
        video_url: formData.video_url.trim() === '' ? null : formData.video_url.trim()
      };

      if (editingHotel) {
        await updateHotel(editingHotel.id, payload);
      } else {
        await createHotel(payload);
      }

      handleCloseModal();
      fetchHotelsAndCities();
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
      fetchHotelsAndCities();
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

  const countByCity = useMemo(() => {
    const map = {};
    hotels.forEach(h => {
      const k = cityKey(h.city);
      map[k] = (map[k] || 0) + 1;
    });
    return map;
  }, [hotels]);

  // Tab kota: satu per kota yang punya hotel (Mekkah/Makkah digabung).
  const cityTabs = useMemo(() => {
    const seen = new Set();
    const list = [];
    cities.forEach(c => {
      const k = cityKey(c);
      if (seen.has(k)) return;
      seen.add(k);
      list.push({ value: k, label: c });
    });
    return list;
  }, [cities]);

  const rows = useMemo(
    () => hotels.filter(h => !cityFilter || cityKey(h.city) === cityFilter),
    [hotels, cityFilter]
  );

  const columns = [
    { header: 'Hotel', key: 'name', sortable: true },
    { header: 'Kota', key: 'city', sortable: true },
    { header: 'Bintang', key: 'star_rating', sortable: true },
    {
      header: 'Jarak ke Masjid',
      key: 'distance_m',
      sortable: true,
      sortFn: (a, b) => (a.distance_m ?? Infinity) - (b.distance_m ?? Infinity),
    },
    { header: 'Video', key: 'video_url' },
    { header: 'Aksi', key: 'aksi' },
  ];

  const renderCell = (row, key) => {
    if (key === 'name') {
      return (
        <div className="flex items-center gap-3 min-w-0">
          <PhotoCell url={row.photo_url} name={row.name} />
          <button
            type="button"
            onClick={() => handleOpenModal(row)}
            className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors"
          >
            {row.name}
          </button>
        </div>
      );
    }

    if (key === 'city') {
      return (
        <span className="inline-flex items-center gap-1.5 text-[13px] font-body text-neutral-700 whitespace-nowrap">
          <MapPin className="w-3.5 h-3.5 text-neutral-400" strokeWidth={1.75} />
          {row.city}
        </span>
      );
    }

    if (key === 'star_rating') {
      if (row.star_rating === null || row.star_rating === undefined) {
        return <span className="text-xs text-neutral-400 font-body">-</span>;
      }
      return (
        <span className="inline-flex items-center gap-1 text-[13px] font-body font-medium text-neutral-800">
          <Star className="w-3.5 h-3.5 fill-warning-400 text-warning-400" strokeWidth={1.5} />
          {row.star_rating}
        </span>
      );
    }

    if (key === 'distance_m') {
      if (row.distance_m === null || row.distance_m === undefined) {
        return <span className="text-xs text-neutral-400 font-body">-</span>;
      }
      return <span className="text-[13px] font-body text-neutral-700 whitespace-nowrap">±{row.distance_m} m</span>;
    }

    if (key === 'video_url') {
      if (!row.video_url) return <span className="text-xs text-neutral-400 font-body">-</span>;
      return (
        <a
          href={row.video_url}
          target="_blank"
          rel="noopener noreferrer"
          onClick={(e) => e.stopPropagation()}
          className="inline-flex items-center gap-1 text-xs font-medium font-body text-primary-600 hover:text-primary-700 hover:underline"
          title={row.video_url}
        >
          <Play className="w-3 h-3 fill-current" />
          Tonton
        </a>
      );
    }

    if (key === 'aksi') {
      const menuItems = [
        { label: 'Edit', icon: Pencil, tooltip: 'Edit Hotel', onClick: () => handleOpenModal(row) },
        { label: 'Hapus', icon: Trash2, danger: true, tooltip: 'Hapus Hotel', onClick: () => confirmDelete(row.id) },
      ];
      return <ActionMenu items={menuItems} align="right" />;
    }

    return row[key];
  };

  const filterTabs = [{ value: '', label: 'Semua', count: hotels.length }, ...cityTabs.map(c => ({ ...c, count: countByCity[c.value] || 0 }))];

  return (
    <div className="space-y-6">
      <PageHeader
        title="Kelola Hotel"
        actionLabel="+ Tambah Hotel"
        onAction={() => handleOpenModal()}
      />

      {errorMessage && (
        <Alert variant="error" onClose={() => setErrorMessage(null)}>{errorMessage}</Alert>
      )}

      {isLoading ? (
        <div className="flex justify-center p-8 bg-white rounded-2xl border border-neutral-200 shadow-card"><LoadingSpinner /></div>
      ) : (
        <DataTable
          columns={columns}
          data={rows}
          itemsPerPage={10}
          emptyMessage='Belum ada hotel. Klik "+ Tambah Hotel" untuk menambahkan.'
          searchPlaceholder="Cari nama hotel atau kota..."
          toolbarActions={
            <CustomDropdown
              value={cityFilter}
              onChange={(val) => setCityFilter(val)}
              className="!mb-0 w-52"
              placeholder="Semua Kota"
              options={filterTabs.map(tab => ({
                value: tab.value,
                label: `${tab.value === '' ? 'Semua Kota' : tab.label} (${tab.count})`,
              }))}
            />
          }
          renderCell={renderCell}
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
          <Input
            label="Nama Hotel"
            name="name"
            value={formData.name}
            onChange={handleChange}
            required
            placeholder="Contoh: Hilton Makkah"
          />

          {!isNewCity ? (
            <CustomDropdown
              label="Kota"
              required
              value={formData.city}
              onChange={(val) => {
                if (val === '__new__') {
                  setIsNewCity(true);
                  setFormData(prev => ({ ...prev, city: '' }));
                } else {
                  setFormData(prev => ({ ...prev, city: val }));
                }
              }}
              options={[
                ...cities.map(c => ({ value: c, label: c })),
                { value: '__new__', label: '+ Tambah Kota Baru' }
              ]}
              placeholder="Pilih Kota"
            />
          ) : (
            <div>
              <div className="flex items-center justify-between mb-1">
                <label className="block text-sm font-medium text-neutral-700 font-heading">
                  Kota Baru <span className="text-danger-600">*</span>
                </label>
                <button
                  type="button"
                  onClick={() => {
                    setIsNewCity(false);
                    setFormData(prev => ({ ...prev, city: cities[0] || '' }));
                  }}
                  className="text-xs font-medium text-primary-600 hover:text-primary-700 transition-colors cursor-pointer flex items-center gap-1"
                >
                  ← Pilih dari daftar
                </button>
              </div>
              <Input
                name="city"
                value={formData.city}
                onChange={handleChange}
                required
                placeholder="Contoh: Dubai, Istanbul, Doha"
                className="!mb-0"
              />
            </div>
          )}

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

          <Input
            label="Jarak ke Masjid (meter)"
            type="number"
            name="distance_m"
            value={formData.distance_m}
            onChange={handleChange}
            placeholder="Contoh: 200 (opsional)"
          />

          <FileInput
            label="Foto Hotel (Opsional)"
            value={formData.photo_url}
            previewUrl={localPreview}
            onChange={handleFileChange}
            onRemove={() => {
              setLocalPreview(null);
              setFormData(prev => ({ ...prev, photo_url: '' }));
            }}
            isUploading={isUploading}
            uploadingText="Mengupload foto hotel..."
            placeholder="Pilih file foto hotel (JPG, PNG, WebP)..."
          />

          <Input
            label="Link Video Hotel (Opsional)"
            name="video_url"
            value={formData.video_url}
            onChange={handleChange}
            placeholder="Contoh: https://www.youtube.com/watch?v=... atau https://youtu.be/..."
          />

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
