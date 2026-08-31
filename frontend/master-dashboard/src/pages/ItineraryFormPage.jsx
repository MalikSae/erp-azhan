import React, { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import PageHeader from '../components/ui/PageHeader';
import MetaBox from '../components/ui/MetaBox';
import Card from '../components/ui/Card';
import Button from '../components/ui/Button';
import Input from '../components/ui/Input';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import { getItinerary, createItinerary, updateItinerary } from '../api/itineraries';
import { Calendar, MapPin, Plus, Trash2, Save, Info, CheckCircle2, Clock, X } from 'lucide-react';

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
            title: data.title || '',
            days: (data.days || []).map(d => ({
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

    if (!formData.title.trim()) {
      setFormErrors("Judul itinerary wajib diisi.");
      return;
    }

    setIsSubmitting(true);

    try {
      const payload = {
        title: formData.title.trim(),
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
        <PageHeader title="Itinerary Tidak Ditemukan" onBack={() => navigate('/itineraries')} />
        <Alert variant="error">Data itinerary yang Anda cari tidak ada atau sudah dihapus.</Alert>
      </div>
    );
  }

  if (isLoadingDetail) {
    return (
      <div className="flex justify-center p-12 bg-white rounded-lg border border-neutral-200 shadow-sm">
        <LoadingSpinner />
      </div>
    );
  }

  const totalActivities = formData.days.reduce((acc, d) => acc + (d.activities?.length || 0), 0);

  return (
    <div className="space-y-6">
      <PageHeader 
        title={isEditMode ? "Edit Master Itinerary" : "Tambah Itinerary Baru"}
        subtitle={isEditMode ? `Perbarui rangkaian jadwal perjalanan untuk ${formData.title || "itinerary"}` : "Penyusunan jadwal aktivitas perjalanan umroh hari demi hari"}
        onBack={() => navigate('/itineraries')}
      />

      {formErrors && (
        <Alert variant="error">{formErrors}</Alert>
      )}

      <form onSubmit={handleSubmit}>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
          
          {/* KOLOM KIRI: Konten Utama Form */}
          <div className="lg:col-span-2 space-y-6">
            
            {/* MetaBox: Informasi Dasar */}
            <MetaBox
              title="Informasi Itinerary"
              subtitle="Judul dan identitas master itinerary"
              icon={<Calendar size={18} className="text-neutral-700" />}
            >
              <div>
                <Input 
                  label="Judul Itinerary"
                  value={formData.title}
                  onChange={handleTitleChange}
                  required
                  placeholder="Contoh: Paket Umroh Reguler 09 Hari"
                  className="!mb-0"
                />
              </div>
            </MetaBox>

            {/* MetaBox: Rangkaian Perjalanan Hari demi Hari */}
            <MetaBox
              title="Rangkaian Perjalanan (Hari demi Hari)"
              subtitle="Aktivitas dan lokasi setiap hari dalam paket"
              icon={<MapPin size={18} className="text-neutral-700" />}
              headerActions={
                <Button 
                  type="button" 
                  variant="secondary" 
                  size="sm" 
                  onClick={handleAddDay}
                  className="flex items-center gap-1.5 text-xs"
                >
                  <Plus size={14} />
                  <span>Tambah Hari</span>
                </Button>
              }
            >
              {formData.days.length === 0 ? (
                <div className="text-center py-8 px-4 bg-neutral-50 rounded-xl border border-dashed border-neutral-300">
                  <Clock size={32} className="mx-auto text-neutral-400 mb-2" />
                  <p className="text-sm font-medium text-neutral-700 font-heading">Belum ada hari perjalanan</p>
                  <p className="text-xs text-neutral-500 font-body mt-1">
                    Klik tombol "Tambah Hari" untuk mulai menyusun jadwal hari pertama perjalanan.
                  </p>
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    onClick={handleAddDay}
                    className="mt-4 inline-flex items-center gap-1.5 text-xs"
                  >
                    <Plus size={14} />
                    <span>Tambah Hari Pertama</span>
                  </Button>
                </div>
              ) : (
                <div className="space-y-5">
                  {formData.days.map((day, dayIndex) => (
                    <div key={dayIndex} className="bg-neutral-50 rounded-xl border border-neutral-200/90 p-5 space-y-4">
                      <div className="flex items-center justify-between pb-3 border-b border-neutral-200/80">
                        <div className="flex items-center gap-2">
                          <span className="w-6 h-6 rounded-full bg-primary-500 text-neutral-900 text-xs font-bold font-mono flex items-center justify-center">
                            {dayIndex + 1}
                          </span>
                          <h4 className="font-semibold text-neutral-900 font-heading text-sm">
                            Hari ke-{dayIndex + 1}
                          </h4>
                        </div>
                        <button 
                          type="button" 
                          onClick={() => handleRemoveDay(dayIndex)}
                          className="w-7 h-7 flex items-center justify-center rounded-lg text-neutral-400 hover:text-danger-600 hover:bg-danger-50 transition-colors cursor-pointer"
                          title="Hapus Hari"
                          aria-label={`Hapus Hari ke-${dayIndex + 1}`}
                        >
                          <X size={16} />
                        </button>
                      </div>
                      
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <Input 
                          label="Judul Hari"
                          value={day.title}
                          onChange={(e) => handleDayChange(dayIndex, 'title', e.target.value)}
                          placeholder="mis. Keberangkatan & Tiba di Jeddah"
                          className="!mb-0"
                        />
                        <Input 
                          label="Lokasi / Kota"
                          value={day.location}
                          onChange={(e) => handleDayChange(dayIndex, 'location', e.target.value)}
                          placeholder="mis. Jakarta - Madinah"
                          className="!mb-0"
                        />
                      </div>

                      <div className="pt-2">
                        <label className="block text-xs font-semibold text-neutral-700 font-heading mb-2.5">
                          Aktivitas & Rencana Acara
                        </label>
                        <div className="space-y-2.5">
                          {day.activities.map((act, actIndex) => (
                            <div key={actIndex} className="flex flex-col sm:flex-row gap-2 sm:items-center">
                              <div className="w-full sm:w-28 shrink-0">
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
                                  className="w-full rounded-md border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 font-mono text-center transition-colors"
                                />
                              </div>
                              <div className="flex-1">
                                <input 
                                  type="text"
                                  value={act.text}
                                  onChange={(e) => handleActivityChange(dayIndex, actIndex, 'text', e.target.value)}
                                  placeholder="Deskripsi agenda aktivitas..."
                                  required
                                  className="w-full rounded-md border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 font-body transition-colors"
                                />
                              </div>
                              <button 
                                type="button" 
                                onClick={() => handleRemoveActivity(dayIndex, actIndex)}
                                className="w-8 h-8 flex items-center justify-center text-neutral-400 hover:text-danger-600 hover:bg-danger-50 rounded-lg transition-colors p-1 shrink-0"
                                title="Hapus aktivitas"
                              >
                                <Trash2 size={15} />
                              </button>
                            </div>
                          ))}

                          <div className="pt-1">
                            <Button 
                              type="button" 
                              variant="ghost"
                              size="sm"
                              onClick={() => handleAddActivity(dayIndex)}
                              className="text-primary-600 hover:text-primary-700 hover:bg-primary-50 text-xs flex items-center gap-1"
                            >
                              <Plus size={13} />
                              <span>Tambah Baris Aktivitas</span>
                            </Button>
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}

                  <div className="mt-4 flex justify-center">
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      onClick={handleAddDay}
                      className="w-full sm:w-auto flex items-center justify-center gap-1.5 text-xs"
                    >
                      <Plus size={14} />
                      <span>Tambah Hari Berikutnya</span>
                    </Button>
                  </div>
                </div>
              )}
            </MetaBox>

          </div>

          {/* KOLOM KANAN: Sidebar Aksi & Informasi */}
          <div className="space-y-6 lg:sticky lg:top-6">
            
            {/* Panel Aksi & Simpan */}
            <MetaBox 
              title="Aksi & Simpan" 
              subtitle="Penyimpanan master itinerary"
              icon={<Save size={18} className="text-neutral-700" />}
            >
              <div className="space-y-3.5">
                <div className="p-3 bg-neutral-50 rounded-xl border border-neutral-200/80 text-xs text-neutral-600 space-y-1.5 font-body">
                  <div className="flex items-center gap-1.5 text-neutral-900 font-semibold">
                    <CheckCircle2 size={14} className="text-success-600" />
                    <span>Verifikasi Data</span>
                  </div>
                  <p>Master itinerary ini dapat dihubungkan ke berbagai paket jadwal keberangkatan umroh.</p>
                </div>

                <div className="space-y-2 pt-1">
                  <Button 
                    type="submit" 
                    variant="primary" 
                    disabled={isSubmitting} 
                    className="w-full justify-center shadow-sm"
                  >
                    {isSubmitting ? "Menyimpan..." : (isEditMode ? "Perbarui Itinerary" : "Simpan Itinerary")}
                  </Button>
                  <Button 
                    type="button" 
                    variant="ghost" 
                    onClick={() => navigate('/itineraries')} 
                    className="w-full justify-center text-neutral-600 hover:text-neutral-900"
                  >
                    Batal
                  </Button>
                </div>
              </div>
            </MetaBox>

            {/* Panel Ringkasan */}
            <MetaBox 
              title="Ringkasan Perjalanan" 
              subtitle="Statistik jadwal itinerary"
              icon={<Info size={18} className="text-neutral-700" />}
            >
              <div className="space-y-2.5 text-xs font-body text-neutral-600">
                <div className="flex items-center justify-between py-1 border-b border-neutral-100">
                  <span>Total Durasi Hari:</span>
                  <span className="font-bold text-neutral-900 font-mono text-sm">
                    {formData.days.length} Hari
                  </span>
                </div>
                <div className="flex items-center justify-between py-1">
                  <span>Total Agenda Aktivitas:</span>
                  <span className="font-bold text-neutral-900 font-mono text-sm">
                    {totalActivities} Agenda
                  </span>
                </div>
              </div>
            </MetaBox>

          </div>

        </div>
      </form>
    </div>
  );
};

export default ItineraryFormPage;
