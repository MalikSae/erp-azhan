import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { createBooking } from "../api/bookings";
import { listJamaah } from "../api/jamaah";
import { listSchedulesAdmin } from "../api/schedules";
import { listBrands } from "../api/brands";
import PageHeader from "../components/ui/PageHeader";
import MetaBox from "../components/ui/MetaBox";
import CustomDropdown from "../components/ui/CustomDropdown";
import Button from "../components/ui/Button";
import Alert from "../components/ui/Alert";
import LoadingSpinner from "../components/ui/LoadingSpinner";

export const BookingFormPage = ({ showBrandColumn = false }) => {
  const navigate = useNavigate();

  const [loadingData, setLoadingData] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState(null);

  const [jamaahList, setJamaahList] = useState([]);
  const [scheduleList, setScheduleList] = useState([]);
  const [brandsMap, setBrandsMap] = useState({});

  const [formData, setFormData] = useState({
    jamaah_id: "",
    schedule_id: "",
    room_type: "Quad"
  });

  useEffect(() => {
    const fetchOptions = async () => {
      try {
        const promises = [
          listJamaah(),
          listSchedulesAdmin({ status: 'published' })
        ];
        if (showBrandColumn) {
          promises.push(listBrands());
        }

        const [jamaahRes, schedRes, brandsRes] = await Promise.all(promises);
        setJamaahList(jamaahRes || []);
        const published = (schedRes || []).filter(s => s.status === 'published');
        setScheduleList(published);

        if (brandsRes) {
          const bMap = {};
          brandsRes.forEach(b => {
            bMap[b.id] = b;
          });
          setBrandsMap(bMap);
        }
      } catch (err) {
        setError("Gagal memuat opsi jamaah/paket.");
      } finally {
        setLoadingData(false);
      }
    };
    fetchOptions();
  }, [showBrandColumn]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!formData.jamaah_id || !formData.schedule_id) {
      setError("Pilih Jamaah dan Paket");
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      const payload = {
        jamaah_id: parseInt(formData.jamaah_id, 10),
        schedule_id: parseInt(formData.schedule_id, 10),
        room_type: formData.room_type
      };
      const res = await createBooking(payload);
      navigate(`/bookings/${res.id}`);
    } catch (err) {
      setError(err.response?.data?.error || "Gagal membuat booking");
      setSubmitting(false);
    }
  };

  if (loadingData) return <div className="p-8 flex justify-center"><LoadingSpinner /></div>;

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <PageHeader 
        title="Booking Baru" 
        onBack={() => navigate(-1)}
      />

      {error && <Alert variant="error">{error}</Alert>}

      <form onSubmit={handleSubmit} className="space-y-6">
        <MetaBox title="Pilih Jamaah dan Paket">
          <div className="space-y-4">
            <CustomDropdown 
              label="Pilih Jamaah"
              name="jamaah_id" 
              value={formData.jamaah_id} 
              onChange={(val) => setFormData(prev => ({ ...prev, jamaah_id: val }))} 
              placeholder="-- Pilih Jamaah --"
              options={jamaahList.map(j => ({
                value: j.id,
                label: `${j.nama_lengkap} ${j.nik ? `(${j.nik})` : ''}`
              }))}
              required
            />

            <CustomDropdown 
              label="Pilih Paket (Hanya Publish)"
              name="schedule_id" 
              value={formData.schedule_id} 
              onChange={(val) => setFormData(prev => ({ ...prev, schedule_id: val }))} 
              placeholder="-- Pilih Paket --"
              options={scheduleList.map(s => {
                const brandName = showBrandColumn && (brandsMap[s.brand_id]?.name || s.brand?.name);
                const brandPrefix = brandName ? `[${brandName}] ` : '';
                return {
                  value: s.id,
                  label: `${brandPrefix}${s.jadwal_nama} - Sisa: ${s.seat_sisa} pax`
                };
              })}
              required
            />

            <CustomDropdown 
              label="Tipe Kamar" 
              name="room_type" 
              value={formData.room_type} 
              onChange={(val) => setFormData(prev => ({ ...prev, room_type: val }))}
              options={[
                { value: "Quad", label: "Quad" },
                { value: "Triple", label: "Triple" },
                { value: "Double", label: "Double" }
              ]}
            />
          </div>
        </MetaBox>

        <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
          <Button type="button" variant="ghost" onClick={() => navigate("/bookings")}>Batal</Button>
          <Button type="submit" variant="primary" disabled={submitting}>
            {submitting ? "Memproses..." : "Buat Booking"}
          </Button>
        </div>
      </form>
    </div>
  );
};

export default BookingFormPage;
