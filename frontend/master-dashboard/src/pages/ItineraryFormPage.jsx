import React, { useState, useEffect } from 'react';
import { useNavigate, useParams, Link } from 'react-router-dom';
import PageHeader from '../components/ui/PageHeader';
import Card from '../components/ui/Card';
import Button from '../components/ui/Button';
import FormField from '../components/ui/FormField';
import Input from '../components/ui/Input';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import { getItinerary, createItinerary, updateItinerary } from '../api/itineraries';

const initialForm = {
  title: '',
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

  useEffect(() => {
    if (isEditMode) {
      const fetchDetail = async () => {
        setIsLoadingDetail(true);
        setFormErrors(null);
        setNotFound(false);
        try {
          const data = await getItinerary(id);
          setFormData({
            title: data.title,
            days: data.days.map(d => ({
              title: d.title || '',
              location: d.location || '',
              activities: d.activities || []
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

  const handleTitleChange = (e) => {
    setFormData(prev => ({ ...prev, title: e.target.value }));
  };

  const handleAddDay = () => {
    setFormData(prev => ({
      ...prev,
      days: [
        ...prev.days,
        { title: '', location: '', activities: [{ time: '', text: '' }] }
      ]
    }));
  };

  const handleRemoveDay = (dayIndex) => {
    setFormData(prev => {
      const newDays = [...prev.days];
      newDays.splice(dayIndex, 1);
      return { ...prev, days: newDays };
    });
  };

  const handleDayChange = (dayIndex, field, value) => {
    setFormData(prev => {
      const newDays = [...prev.days];
      newDays[dayIndex] = { ...newDays[dayIndex], [field]: value };
      return { ...prev, days: newDays };
    });
  };

  const handleAddActivity = (dayIndex) => {
    setFormData(prev => {
      const newDays = [...prev.days];
      newDays[dayIndex] = {
        ...newDays[dayIndex],
        activities: [...newDays[dayIndex].activities, { time: '', text: '' }]
      };
      return { ...prev, days: newDays };
    });
  };

  const handleRemoveActivity = (dayIndex, actIndex) => {
    setFormData(prev => {
      const newDays = [...prev.days];
      const newActivities = [...newDays[dayIndex].activities];
      newActivities.splice(actIndex, 1);
      newDays[dayIndex] = { ...newDays[dayIndex], activities: newActivities };
      return { ...prev, days: newDays };
    });
  };

  const handleActivityChange = (dayIndex, actIndex, field, value) => {
    setFormData(prev => {
      const newDays = [...prev.days];
      const newActivities = [...newDays[dayIndex].activities];
      newActivities[actIndex] = { ...newActivities[actIndex], [field]: value };
      newDays[dayIndex] = { ...newDays[dayIndex], activities: newActivities };
      return { ...prev, days: newDays };
    });
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setFormErrors(null);
    setIsSubmitting(true);

    try {
      const payload = {
        title: formData.title,
        days: formData.days
      };

      if (isEditMode) {
        await updateItinerary(id, payload);
      } else {
        await createItinerary(payload);
      }
      navigate('/itineraries');
    } catch (error) {
      const msg = error.response?.data?.error || "Terjadi kesalahan, coba lagi.";
      setFormErrors(msg);
    } finally {
      setIsSubmitting(false);
    }
  };

  if (notFound) {
    return (
      <div className="space-y-6">
        <PageHeader title="Itinerary Tidak Ditemukan" />
        <Card className="p-8 text-center">
          <p className="text-neutral-600 mb-4 font-body">Data itinerary yang Anda cari tidak ada atau sudah dihapus.</p>
          <Button onClick={() => navigate('/itineraries')} variant="primary">
            Kembali ke Daftar
          </Button>
        </Card>
      </div>
    );
  }

  if (isLoadingDetail) {
    return (
      <div className="space-y-6">
        <PageHeader title={isEditMode ? "Edit Itinerary" : "Tambah Itinerary"} />
        <Card className="p-8 flex justify-center">
          <LoadingSpinner />
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <button 
          onClick={() => navigate('/itineraries')}
          className="text-sm text-neutral-500 hover:text-neutral-800 flex items-center gap-1 mb-4 font-medium transition-colors"
        >
          <span>&larr;</span> Kembali ke daftar
        </button>
        <PageHeader title={isEditMode ? "Edit Itinerary" : "Tambah Itinerary"} />
      </div>

      <Card className="max-w-4xl">
        <form onSubmit={handleSubmit} className="p-6 space-y-8">
          <Input 
            label="Judul Itinerary"
            value={formData.title}
            onChange={handleTitleChange}
            required
            placeholder="Contoh: Paket Umroh Plus Turki 12 Hari"
            className="!mb-0"
          />

          <div className="space-y-6">
            <h3 className="font-heading text-lg font-semibold text-neutral-800 border-b border-neutral-200 pb-2">Jadwal Perjalanan</h3>
            {formData.days.map((day, dayIndex) => (
              <div key={dayIndex} className="bg-neutral-50 rounded-md border border-neutral-200 p-5 space-y-5">
                <div className="flex items-center justify-between">
                  <h4 className="font-semibold text-neutral-800 font-heading">Hari {dayIndex + 1}</h4>
                  <button 
                    type="button" 
                    onClick={() => handleRemoveDay(dayIndex)}
                    className="text-sm text-danger-600 hover:text-danger-700 font-medium bg-transparent px-3 py-1 rounded transition-colors hover:bg-danger-50"
                  >
                    Hapus Hari
                  </button>
                </div>
                
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                  <Input 
                    label="Judul Hari"
                    value={day.title}
                    onChange={(e) => handleDayChange(dayIndex, 'title', e.target.value)}
                    placeholder="mis. Hari 1 - Keberangkatan"
                    className="!mb-0"
                  />
                  <Input 
                    label="Lokasi"
                    value={day.location}
                    onChange={(e) => handleDayChange(dayIndex, 'location', e.target.value)}
                    placeholder="mis. Jakarta - Madinah"
                    className="!mb-0"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-neutral-700 mb-2 font-body">Aktivitas</label>
                  <div>
                    {day.activities.map((act, actIndex) => (
                      <div key={actIndex} className="flex flex-col sm:flex-row gap-3 sm:items-center mb-4">
                        <div className="w-full sm:w-24 shrink-0">
                          <input
                            type="text" 
                            value={act.time}
                            onChange={(e) => {
                              let val = e.target.value.replace(/[^\d:]/g, '');
                              let parts = val.split(':');
                              if (parts.length > 2) {
                                val = parts[0] + ':' + parts.slice(1).join('');
                                parts = val.split(':');
                              }
                              
                              if (parts[0].length > 2) {
                                parts[1] = parts[0].substring(2) + (parts[1] || '');
                                parts[0] = parts[0].substring(0, 2);
                              }
                              if (parts[0].length === 2 && parseInt(parts[0], 10) > 23) {
                                parts[0] = '23';
                              }
                              
                              if (parts.length > 1) {
                                if (parts[1].length > 2) parts[1] = parts[1].substring(0, 2);
                                if (parts[1].length === 2 && parseInt(parts[1], 10) > 59) {
                                  parts[1] = '59';
                                }
                              }
                              
                              val = parts.length > 1 ? `${parts[0]}:${parts[1]}` : parts[0];
                              handleActivityChange(dayIndex, actIndex, 'time', val);
                            }}
                            placeholder="09:00"
                            className="w-full rounded-md border border-neutral-300 px-3 py-2 text-sm text-neutral-900 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 font-body transition-colors text-center"
                          />
                        </div>
                        <div className="flex-grow">
                          <input 
                            type="text"
                            value={act.text}
                            onChange={(e) => handleActivityChange(dayIndex, actIndex, 'text', e.target.value)}
                            placeholder="Deskripsi aktivitas"
                            required
                            className="w-full rounded-md border border-neutral-300 px-3 py-2 text-sm text-neutral-900 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 font-body transition-colors"
                          />
                        </div>
                        <button 
                          type="button" 
                          onClick={() => handleRemoveActivity(dayIndex, actIndex)}
                          className="text-neutral-400 hover:text-danger-600 transition-colors p-1 shrink-0"
                          title="Hapus aktivitas"
                        >
                          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
                          </svg>
                        </button>
                      </div>
                    ))}
                    <Button 
                      type="button" 
                      variant="ghost"
                      size="sm"
                      onClick={() => handleAddActivity(dayIndex)}
                      className="text-primary-600 hover:text-primary-700 hover:bg-primary-50 -ml-2"
                    >
                      <span className="mr-1">+</span> Tambah Aktivitas
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>

          <div>
            <Button type="button" variant="secondary" onClick={handleAddDay}>
              + Tambah Hari
            </Button>
          </div>

          {formErrors && (
            <Alert variant="error">{formErrors}</Alert>
          )}

          <div className="pt-6 flex justify-end space-x-3 border-t border-neutral-200">
            <Button type="button" variant="ghost" onClick={() => navigate('/itineraries')} disabled={isSubmitting}>
              Batal
            </Button>
            <Button type="submit" variant="primary" isLoading={isSubmitting}>
              {isEditMode ? "Update" : "Simpan"}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
};

export default ItineraryFormPage;
