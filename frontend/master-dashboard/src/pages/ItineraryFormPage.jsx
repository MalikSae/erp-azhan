import React, { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import PageHeader from 'shared/ui/PageHeader';
import Button from 'shared/ui/Button';
import Input from 'shared/ui/Input';
import Textarea from 'shared/ui/Textarea';
import CustomDropdown from 'shared/ui/CustomDropdown';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import FormLayout, { FormSection, SidebarSection, FormAside } from 'shared/ui/FormLayout';
import { getItinerary, createItinerary, updateItinerary } from '../api/itineraries';
import { listCategories } from '../api/categories';

const initialForm = {
  title: '',
  category_id: '',
  days: []
};

// Format mengikuti dokumen operasional: per hari ada rute dan poin kegiatan (tanpa jam).
const emptyActivity = () => ({ time: '', text: '' });
const emptyDay = () => ({ title: '', location: '', activities: [emptyActivity()] });

const AddButton = ({ onClick, children }) => (
  <button
    type="button"
    onClick={onClick}
    className="w-full py-2 px-4 rounded-md border border-dashed border-neutral-300 hover:border-primary-400 hover:bg-neutral-50 text-neutral-700 hover:text-primary-600 font-medium text-xs flex items-center justify-center gap-1.5 transition-colors cursor-pointer"
  >
    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 4v16m8-8H4" />
    </svg>
    <span>{children}</span>
  </button>
);

const RemoveButton = ({ onClick, title }) => (
  <button
    type="button"
    onClick={onClick}
    title={title}
    aria-label={title}
    className="h-10 px-3 inline-flex items-center text-danger-600 hover:text-danger-700 hover:bg-danger-50 rounded-md border border-neutral-200 transition-colors cursor-pointer shrink-0"
  >
    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
    </svg>
  </button>
);

const ItineraryFormPage = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const isEditMode = Boolean(id);

  const [formData, setFormData] = useState(initialForm);
  const [formErrors, setFormErrors] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isLoadingDetail, setIsLoadingDetail] = useState(isEditMode);
  const [notFound, setNotFound] = useState(false);
  const [categories, setCategories] = useState([]);

  useEffect(() => {
    listCategories().then(data => setCategories((data || []).filter(c => c.is_active))).catch(() => setCategories([]));
  }, []);

  useEffect(() => {
    if (isEditMode) {
      const fetchDetail = async () => {
        setIsLoadingDetail(true);
        setFormErrors(null);
        setNotFound(false);
        try {
          const data = await getItinerary(id);
          setFormData({
            title: data.title || '',
            category_id: data.category_id ? String(data.category_id) : '',
            days: (data.days || []).map(d => ({
              title: d.title || '',
              location: d.location || d.title || '',
              // Format lama memakai jam per kegiatan; kini jam (jika ada) menjadi bagian teks.
              activities: (d.activities || []).map(act => ({
                time: '',
                text: [act.time, act.text].filter(Boolean).join(' ').trim()
              }))
            }))
          });
        } catch (error) {
          if (error.response?.status === 404) {
            setNotFound(true);
          } else {
            setFormErrors(error.response?.data?.error || "Gagal memuat detail itinerary.");
          }
        } finally {
          setIsLoadingDetail(false);
        }
      };
      fetchDetail();
    } else {
      setFormData(initialForm);
    }
  }, [id, isEditMode]);

  const updateDay = (dayIndex, updater) => {
    setFormData(prev => {
      const days = [...prev.days];
      days[dayIndex] = updater(days[dayIndex]);
      return { ...prev, days };
    });
  };

  const handleAddDay = () => setFormData(prev => ({ ...prev, days: [...prev.days, emptyDay()] }));

  const handleRemoveDay = (dayIndex) => {
    setFormData(prev => ({ ...prev, days: prev.days.filter((_, i) => i !== dayIndex) }));
  };

  const handleDayChange = (dayIndex, field, value) => {
    updateDay(dayIndex, day => ({ ...day, [field]: value }));
  };

  const handleAddActivity = (dayIndex) => {
    updateDay(dayIndex, day => ({ ...day, activities: [...day.activities, emptyActivity()] }));
  };

  const handleRemoveActivity = (dayIndex, actIndex) => {
    updateDay(dayIndex, day => ({ ...day, activities: day.activities.filter((_, i) => i !== actIndex) }));
  };

  const handleActivityChange = (dayIndex, actIndex, field, value) => {
    updateDay(dayIndex, day => ({
      ...day,
      activities: day.activities.map((act, i) => (i === actIndex ? { ...act, [field]: value } : act))
    }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setFormErrors(null);

    if (!formData.title.trim()) {
      setFormErrors("Judul itinerary wajib diisi.");
      return;
    }

    setIsSubmitting(true);

    try {
      const payload = {
        title: formData.title.trim(),
        category_id: formData.category_id ? parseInt(formData.category_id, 10) : null,
        days: formData.days.map(day => ({
          title: day.title || '',
          location: (day.location || '').trim(),
          activities: day.activities
            .map(act => ({ time: '', text: (act.text || '').trim() }))
            .filter(act => act.text !== '')
        }))
      };

      if (isEditMode) {
        await updateItinerary(id, payload);
      } else {
        await createItinerary(payload);
      }
      navigate('/itineraries', {
        state: { successMessage: `Itinerary "${payload.title}" berhasil ${isEditMode ? 'diperbarui' : 'ditambahkan'}.` },
      });
    } catch (error) {
      setFormErrors(error.response?.data?.error || "Terjadi kesalahan, coba lagi.");
    } finally {
      setIsSubmitting(false);
    }
  };

  if (notFound) {
    return (
      <div className="space-y-6">
        <PageHeader title="Itinerary Tidak Ditemukan" onBack={() => navigate('/itineraries')} />
        <Alert variant="error">Data itinerary yang Anda cari tidak ada atau sudah dihapus.</Alert>
      </div>
    );
  }

  if (isLoadingDetail) {
    return (
      <div className="flex justify-center p-12 bg-white rounded-2xl border border-neutral-200 shadow-card">
        <LoadingSpinner />
      </div>
    );
  }

  const totalActivities = formData.days.reduce((acc, d) => acc + (d.activities?.length || 0), 0);

  return (
    <div className="space-y-6">
      <PageHeader
        title={isEditMode ? 'Edit Itinerary' : 'Tambah Itinerary'}
        onBack={() => navigate('/itineraries')}
      />

      {formErrors && <Alert variant="error" onClose={() => setFormErrors(null)}>{formErrors}</Alert>}

      <form onSubmit={handleSubmit}>
        <FormLayout>
          <FormSection id="sec-dasar" title="Informasi Dasar">
            <Input
              label="Judul Itinerary"
              value={formData.title}
              onChange={(e) => setFormData(prev => ({ ...prev, title: e.target.value }))}
              required
              placeholder="mis. Umroh Reguler 9 Hari"
              className="!mb-0"
            />
          </FormSection>

          <FormSection id="sec-hari" title="Rangkaian Perjalanan">
            <div className="space-y-6">
              {formData.days.length === 0 && (
                <p className="text-sm text-neutral-500 font-body">
                  Belum ada hari perjalanan. Tambahkan hari pertama untuk mulai menyusun jadwal.
                </p>
              )}

              {formData.days.map((day, dayIndex) => {
                return (
                  <div key={dayIndex} className="space-y-4 border-t border-neutral-200 pt-5 first:border-t-0 first:pt-0">
                    <div className="flex items-center justify-between">
                      <h4 className="font-heading text-sm font-semibold text-neutral-900">Hari ke-{dayIndex + 1}</h4>
                      <button
                        type="button"
                        onClick={() => handleRemoveDay(dayIndex)}
                        className="text-xs font-medium text-neutral-500 hover:text-danger-600 transition-colors cursor-pointer"
                      >
                        Hapus hari
                      </button>
                    </div>

                    <Input
                      label="Lokasi"
                      value={day.location}
                      onChange={(e) => handleDayChange(dayIndex, 'location', e.target.value)}
                      placeholder="mis. Makkah – Madinah"
                      required
                      className="!mb-0"
                    />

                    <div className="space-y-2">
                      <span className="block min-h-5 text-[13px] font-medium leading-5 text-neutral-800 font-body">Kegiatan</span>
                      {day.activities.map((act, actIndex) => (
                        <div key={actIndex} className="flex items-start gap-3">
                          <div className="min-w-0 flex-1">
                            <Textarea
                              value={act.text}
                              onChange={(e) => handleActivityChange(dayIndex, actIndex, 'text', e.target.value)}
                              placeholder="mis. Sarapan pagi lalu berkumpul di lobby"
                              rows={2}
                              className="!mb-0"
                            />
                          </div>
                          <RemoveButton
                            title="Hapus poin kegiatan"
                            onClick={() => handleRemoveActivity(dayIndex, actIndex)}
                          />
                        </div>
                      ))}
                      <AddButton onClick={() => handleAddActivity(dayIndex)}>Tambah Poin Kegiatan</AddButton>
                      <p className="text-xs text-neutral-500 font-body">
                        Satu poin per kotak. Tebalkan kata dengan **tanda bintang dua**.
                      </p>
                    </div>
                  </div>
                );
              })}

              <AddButton onClick={handleAddDay}>
                {formData.days.length === 0 ? 'Tambah Hari Pertama' : 'Tambah Hari'}
              </AddButton>
            </div>
          </FormSection>

          <FormAside>
            <SidebarSection title="Simpan">
              <div className="space-y-3">
                <p className="text-xs text-neutral-500 font-body">
                  Itinerary ini bisa dipilih pada banyak paket keberangkatan.
                </p>
                <Button
                  type="submit"
                  variant="primary"
                  isLoading={isSubmitting}
                  disabled={isSubmitting}
                  className="w-full justify-center"
                >
                  {isSubmitting ? 'Menyimpan...' : (isEditMode ? 'Perbarui Itinerary' : 'Simpan Itinerary')}
                </Button>
              </div>
            </SidebarSection>

            <SidebarSection title="Kategori">
              <CustomDropdown
                className="!mb-0"
                value={formData.category_id}
                onChange={(val) => setFormData(prev => ({ ...prev, category_id: val }))}
                placeholder="-- Semua kategori --"
                options={[
                  { value: '', label: '-- Semua kategori --' },
                  ...categories.map(c => ({ value: String(c.id), label: c.name })),
                ]}
              />
              <p className="mt-2 text-xs text-neutral-500 font-body">
                Itinerary muncul sebagai pilihan pada paket dengan kategori ini.
              </p>
            </SidebarSection>

            <SidebarSection title="Ringkasan">
              <dl className="space-y-2 text-xs font-body">
                <div className="flex items-center justify-between">
                  <dt className="text-neutral-500">Durasi</dt>
                  <dd className="text-sm font-semibold text-neutral-900">{formData.days.length} hari</dd>
                </div>
                <div className="flex items-center justify-between">
                  <dt className="text-neutral-500">Total poin kegiatan</dt>
                  <dd className="text-sm font-semibold text-neutral-900">{totalActivities}</dd>
                </div>
              </dl>
            </SidebarSection>
          </FormAside>
        </FormLayout>
      </form>
    </div>
  );
};

export default ItineraryFormPage;
