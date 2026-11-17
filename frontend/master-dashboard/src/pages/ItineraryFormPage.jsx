import React, { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import PageHeader from 'shared/ui/PageHeader';
import Button from 'shared/ui/Button';
import Input from 'shared/ui/Input';
import CustomDropdown from 'shared/ui/CustomDropdown';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import FormLayout, { FormSection, SidebarSection, FormAside } from 'shared/ui/FormLayout';
import { getItinerary, createItinerary, updateItinerary } from '../api/itineraries';
import { listCategories } from '../api/categories';
import ItineraryDaysEditor, { normalizeDays, daysToPayload } from '../components/ItineraryDaysEditor';

const initialForm = {
  title: '',
  category_id: '',
  days: []
};

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
            days: normalizeDays(data.days)
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
        days: daysToPayload(formData.days)
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
      setFormErrors(error.response?.data?.error || "Data gagal disimpan. Coba lagi.");
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
      <LoadingSpinner variant="card" />
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
            <ItineraryDaysEditor days={formData.days} onChange={(days) => setFormData(prev => ({ ...prev, days }))} />
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
