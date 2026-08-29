import React, { useState, useEffect, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { Plus, Trash2, Users, AlertCircle, Info } from "lucide-react";
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
import Toggle from "../components/ui/Toggle";
import Badge from "../components/ui/Badge";
import Modal from "../components/ui/Modal";

const formatRupiah = (val) => {
  if (val === null || val === undefined || isNaN(Number(val))) return "-";
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0
  }).format(Number(val));
};

export const BookingFormPage = ({ showBrandColumn = false }) => {
  const navigate = useNavigate();

  const [loadingData, setLoadingData] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [serverError, setServerError] = useState(null);
  const [clientErrors, setClientErrors] = useState({});

  const [jamaahList, setJamaahList] = useState([]);
  const [scheduleList, setScheduleList] = useState([]);
  const [brandsMap, setBrandsMap] = useState({});

  const [selectedScheduleId, setSelectedScheduleId] = useState("");
  const [picJamaahId, setPicJamaahId] = useState("");
  const [paxList, setPaxList] = useState([
    { id: 1, jamaah_id: "", is_infant: false, room_type: "" }
  ]);

  // Modal konfirmasi cascade delete saat menghapus pax reguler terakhir yang memiliki infant
  const [cascadeDeleteTarget, setCascadeDeleteTarget] = useState(null);

  useEffect(() => {
    const fetchOptions = async () => {
      try {
        const promises = [
          listJamaah(),
          listSchedulesAdmin({ status: "published" })
        ];
        if (showBrandColumn) {
          promises.push(listBrands());
        }

        const [jamaahRes, schedRes, brandsRes] = await Promise.all(promises);
        setJamaahList(jamaahRes || []);
        const published = (schedRes || []).filter((s) => s.status === "published");
        setScheduleList(published);

        if (brandsRes) {
          const bMap = {};
          brandsRes.forEach((b) => {
            bMap[b.id] = b;
          });
          setBrandsMap(bMap);
        }
      } catch (err) {
        setServerError("Gagal memuat opsi jamaah/paket.");
      } finally {
        setLoadingData(false);
      }
    };
    fetchOptions();
  }, [showBrandColumn]);

  // Cari objek schedule terpilih
  const selectedSchedule = useMemo(() => {
    return scheduleList.find((s) => String(s.id) === String(selectedScheduleId)) || null;
  }, [scheduleList, selectedScheduleId]);

  // Lookup map jamaah untuk label cepat
  const jamaahMap = useMemo(() => {
    const map = {};
    jamaahList.forEach((j) => {
      map[j.id] = j;
    });
    return map;
  }, [jamaahList]);

  // Daftar opsi PIC: HANYA jamaah dari pax REGULER yang sudah terisi jamaah_id-nya
  const picOptions = useMemo(() => {
    const regularPax = paxList.filter((p) => !p.is_infant && Boolean(p.jamaah_id));
    const uniqueIds = Array.from(new Set(regularPax.map((p) => p.jamaah_id)));
    return uniqueIds.map((id) => {
      const j = jamaahMap[id];
      return {
        value: id,
        label: j ? `${j.nama_lengkap} ${j.nik ? `(${j.nik})` : ""}` : `Jamaah #${id}`
      };
    });
  }, [paxList, jamaahMap]);

  // Auto-sync PIC: pertahankan jika masih valid (ada di regular pax), re-kalkulasi ke pax reguler pertama jika invalid
  useEffect(() => {
    const isCurrentPicValid = picOptions.some((opt) => String(opt.value) === String(picJamaahId));
    if (isCurrentPicValid) {
      return;
    }

    const firstRegularPax = paxList.find((p) => !p.is_infant && Boolean(p.jamaah_id));
    if (firstRegularPax) {
      setPicJamaahId(String(firstRegularPax.jamaah_id));
    } else {
      setPicJamaahId("");
    }
  }, [picOptions, paxList, picJamaahId]);

  // Hitung harga per baris pax
  const calculatePaxPrice = (pax) => {
    if (!selectedSchedule) return null;
    if (pax.is_infant) {
      return selectedSchedule.harga_infant != null ? Number(selectedSchedule.harga_infant) : 0;
    }
    if (!pax.room_type) return null;
    const key = `harga_${pax.room_type.toLowerCase()}`;
    return selectedSchedule[key] != null ? Number(selectedSchedule[key]) : 0;
  };

  // Ringkasan Kuota & Total Harga
  const regularPaxCount = useMemo(() => {
    return paxList.filter((p) => !p.is_infant).length;
  }, [paxList]);

  const infantPaxCount = useMemo(() => {
    return paxList.filter((p) => p.is_infant).length;
  }, [paxList]);

  const seatSisa = selectedSchedule ? Number(selectedSchedule.seat_sisa) : 0;
  const isOverQuota = selectedSchedule ? regularPaxCount > seatSisa : false;

  const totalHarga = useMemo(() => {
    if (!selectedSchedule) return 0;
    return paxList.reduce((sum, p) => {
      const price = calculatePaxPrice(p);
      return sum + (price || 0);
    }, 0);
  }, [paxList, selectedSchedule]);

  // Validasi form kelengkapan untuk disable tombol submit
  const hasEmptyPaxJamaah = paxList.some((p) => !p.jamaah_id);
  const hasEmptyPaxRoom = paxList.some((p) => !p.is_infant && !p.room_type);
  const isSubmitDisabled =
    submitting ||
    isOverQuota ||
    !selectedScheduleId ||
    !picJamaahId ||
    hasEmptyPaxJamaah ||
    hasEmptyPaxRoom ||
    regularPaxCount === 0;

  // Repeater handlers
  const handleAddPax = () => {
    const nextId = paxList.length > 0 ? Math.max(...paxList.map((p) => p.id)) + 1 : 1;
    setPaxList((prev) => [
      ...prev,
      { id: nextId, jamaah_id: "", is_infant: false, room_type: "" }
    ]);
  };

  const handleRemovePax = (idToRemove) => {
    const targetIndex = paxList.findIndex((p) => p.id === idToRemove);
    if (targetIndex === -1) return;

    const targetPax = paxList[targetIndex];

    // Jika hanya ada 1 baris di daftar, reset baris tersebut ke kosong
    if (paxList.length === 1) {
      setPaxList([{ id: 1, jamaah_id: "", is_infant: false, room_type: "" }]);
      setClientErrors({});
      return;
    }

    // Jika baris yang dihapus adalah index 0
    if (targetIndex === 0) {
      // Cari reguler lain yang paling awal di index > 0
      const nextRegularIndex = paxList.findIndex((p, idx) => idx > 0 && !p.is_infant);

      if (nextRegularIndex !== -1) {
        // Ditemukan reguler pengganti: promosikan ke index 0, pertahankan urutan relatif sisanya
        const promotedPax = paxList[nextRegularIndex];
        const otherPax = paxList.filter((p, idx) => idx !== 0 && idx !== nextRegularIndex);
        setPaxList([promotedPax, ...otherPax]);
        cleanClientErrorsForId(idToRemove);
        return;
      }

      // Tidak ada reguler lain sama sekali (semua sisa adalah infant)
      const remainingInfants = paxList.filter((p, idx) => idx > 0 && p.is_infant);
      setCascadeDeleteTarget({
        targetPax,
        infantCount: remainingInfants.length
      });
      return;
    }

    // Jika baris yang dihapus adalah index > 0 (bukan index 0): hapus normal
    setPaxList((prev) => prev.filter((p) => p.id !== idToRemove));
    cleanClientErrorsForId(idToRemove);
  };

  const cleanClientErrorsForId = (idToRemove) => {
    setClientErrors((prev) => {
      const next = { ...prev };
      delete next[`pax_${idToRemove}_jamaah`];
      delete next[`pax_${idToRemove}_room`];
      return next;
    });
  };

  const handleConfirmCascadeDelete = () => {
    if (!cascadeDeleteTarget) return;

    // Reset ke 1 baris kosong baru (reguler index 0)
    setPaxList([{ id: 1, jamaah_id: "", is_infant: false, room_type: "" }]);
    setClientErrors({});
    setCascadeDeleteTarget(null);
  };

  const handlePaxChange = (id, field, value) => {
    const actualVal = (value && typeof value === 'object' && 'target' in value) ? value.target.value : value;

    setPaxList((prev) =>
      prev.map((p, idx) => {
        if (p.id !== id) return p;
        // Index 0 selalu reguler permanen
        if (idx === 0 && field === "is_infant") return p;

        const updated = { ...p, [field]: actualVal };
        if (field === "is_infant") {
          if (actualVal === true) {
            updated.room_type = null;
          } else {
            updated.room_type = "";
          }
        }
        return updated;
      })
    );

    setClientErrors((prev) => {
      const next = { ...prev };
      if (field === "jamaah_id") delete next[`pax_${id}_jamaah`];
      if (field === "room_type") delete next[`pax_${id}_room`];
      return next;
    });
  };

  // Submit Handler
  const handleSubmit = async (e) => {
    e.preventDefault();
    setServerError(null);

    const errors = {};
    if (!selectedScheduleId) {
      errors.schedule_id = "Pilih paket / jadwal terlebih dahulu";
    }

    if (paxList.length === 0) {
      errors.pax = "Minimal harus ada 1 pax dalam booking";
    }

    paxList.forEach((p, idx) => {
      if (!p.jamaah_id) {
        errors[`pax_${p.id}_jamaah`] = `Pilih jamaah untuk Pax #${idx + 1}`;
      }
      if (!p.is_infant && !p.room_type) {
        errors[`pax_${p.id}_room`] = `Pilih tipe kamar untuk Pax #${idx + 1}`;
      }
    });

    if (regularPaxCount === 0) {
      errors.pic_jamaah_id = "Minimal 1 pax reguler diperlukan sebagai kontak utama";
    } else if (!picJamaahId) {
      errors.pic_jamaah_id = "Pilih Kontak Utama (PIC) dari daftar jamaah reguler";
    } else {
      const isValidPic = paxList.some((p) => !p.is_infant && String(p.jamaah_id) === String(picJamaahId));
      if (!isValidPic) {
        errors.pic_jamaah_id = "Kontak Utama (PIC) harus merupakan jamaah reguler (bukan infant)";
      }
    }

    if (isOverQuota) {
      errors.quota = `Jumlah pax reguler (${regularPaxCount}) melebihi sisa kuota kursi (${seatSisa})`;
    }

    if (Object.keys(errors).length > 0) {
      setClientErrors(errors);
      return;
    }

    setClientErrors({});
    setSubmitting(true);

    try {
      const payload = {
        schedule_id: parseInt(selectedScheduleId, 10),
        pic_jamaah_id: parseInt(picJamaahId, 10),
        pax: paxList.map((p) => ({
          jamaah_id: parseInt(p.jamaah_id, 10),
          pax_type: p.is_infant ? "infant" : "reguler",
          room_type: p.is_infant ? null : p.room_type
        }))
      };

      const res = await createBooking(payload);
      navigate(`/bookings/${res.id}`);
    } catch (err) {
      setServerError(err.response?.data?.error || "Gagal membuat booking");
      setSubmitting(false);
    }
  };

  if (loadingData) {
    return (
      <div className="p-8 flex justify-center">
        <LoadingSpinner />
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6 pb-12">
      <PageHeader
        title="Booking Baru"
        subtitle="Daftarkan satu atau beberapa jamaah sekaligus dalam satu nomor booking"
        onBack={() => navigate(-1)}
      />

      {serverError && <Alert variant="error">{serverError}</Alert>}
      {clientErrors.quota && <Alert variant="error">{clientErrors.quota}</Alert>}

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* METABOX 1: Pilih Paket & Jadwal */}
        <MetaBox title="Pilih Paket & Jadwal">
          <div className="space-y-4">
            <CustomDropdown
              label="Pilih Paket (Hanya Publish)"
              value={selectedScheduleId}
              onChange={(val) => {
                const actualVal = (val && typeof val === 'object' && 'target' in val) ? val.target.value : val;
                setSelectedScheduleId(actualVal);
                setClientErrors((prev) => {
                  const next = { ...prev };
                  delete next.schedule_id;
                  return next;
                });
              }}
              placeholder="-- Pilih Paket / Jadwal --"
              options={scheduleList.map((s) => {
                const brandName = showBrandColumn && (brandsMap[s.brand_id]?.name || s.brand?.name);
                const brandPrefix = brandName ? `[${brandName}] ` : "";
                return {
                  value: s.id,
                  label: `${brandPrefix}${s.jadwal_nama} - Sisa: ${s.seat_sisa} pax`
                };
              })}
              error={clientErrors.schedule_id}
              required
            />
            {selectedSchedule && (
              <div className="text-xs text-neutral-500 space-y-0.5 bg-neutral-50 p-2.5 rounded-md border border-neutral-200">
                <div className="font-medium text-neutral-700">Daftar Harga Paket:</div>
                <div>Quad: {formatRupiah(selectedSchedule.harga_quad)} | Triple: {formatRupiah(selectedSchedule.harga_triple)} | Double: {formatRupiah(selectedSchedule.harga_double)}</div>
                <div>Infant: {formatRupiah(selectedSchedule.harga_infant || 0)} | Sisa Kuota Kursi: <span className="font-semibold text-neutral-800">{selectedSchedule.seat_sisa} pax</span></div>
              </div>
            )}
          </div>
        </MetaBox>

        {/* METABOX 2: Repeater Daftar Pax */}
        <MetaBox
          title={
            <div className="flex items-center justify-between w-full">
              <span className="flex items-center gap-2">
                <Users className="w-5 h-5 text-primary-600" />
                Daftar Jamaah / Pax ({paxList.length})
              </span>
              <Button
                type="button"
                variant="secondary"
                size="sm"
                onClick={handleAddPax}
                className="flex items-center gap-1.5"
              >
                <Plus className="w-4 h-4" />
                Tambah Pax
              </Button>
            </div>
          }
        >
          <div className="space-y-4">
            {paxList.map((pax, index) => {
              const paxPrice = calculatePaxPrice(pax);
              const isPic = String(pax.jamaah_id) === String(picJamaahId) && Boolean(pax.jamaah_id) && !pax.is_infant;
              const isFirstRow = index === 0;
              const jamaahError = clientErrors[`pax_${pax.id}_jamaah`];
              const roomError = clientErrors[`pax_${pax.id}_room`];

              return (
                <div
                  key={pax.id}
                  className="p-4 rounded-lg border border-neutral-200 bg-white space-y-3 relative hover:border-neutral-300 transition-colors"
                >
                  {/* Header Baris Pax */}
                  <div className="flex items-center justify-between pb-2 border-b border-neutral-100">
                    <div className="flex items-center gap-2">
                      <span className="font-heading font-semibold text-sm text-neutral-900">
                        Pax #{index + 1}
                      </span>
                      {isPic && (
                        <Badge variant="primary" showIcon={false}>
                          PIC Kontak
                        </Badge>
                      )}
                      {pax.is_infant && (
                        <Badge variant="warning" showIcon={false}>
                          Infant (Bayi)
                        </Badge>
                      )}
                    </div>

                    <button
                      type="button"
                      disabled={paxList.length <= 1 && !pax.jamaah_id && !pax.is_infant && !pax.room_type}
                      onClick={() => handleRemovePax(pax.id)}
                      className={`p-1.5 rounded text-neutral-400 hover:text-danger-600 hover:bg-danger-50 transition-colors ${
                        paxList.length <= 1 && !pax.jamaah_id && !pax.is_infant && !pax.room_type
                          ? "opacity-30 cursor-not-allowed pointer-events-none"
                          : ""
                      }`}
                      title={paxList.length <= 1 ? "Reset pax ini" : "Hapus pax ini"}
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>

                  {/* Input Grid */}
                  <div className="grid grid-cols-1 md:grid-cols-12 gap-4 items-start">
                    {/* Pilih Jamaah */}
                    <div className="md:col-span-5">
                      <CustomDropdown
                        label="Pilih Jamaah"
                        value={pax.jamaah_id}
                        onChange={(val) => handlePaxChange(pax.id, "jamaah_id", val)}
                        placeholder="-- Pilih Jamaah --"
                        options={jamaahList.map((j) => ({
                          value: j.id,
                          label: `${j.nama_lengkap} ${j.nik ? `(${j.nik})` : ""}`
                        }))}
                        error={jamaahError}
                        required
                      />
                    </div>

                    {/* Toggle Infant */}
                    <div className="md:col-span-2 flex flex-col justify-start">
                      {!isFirstRow && (
                        <>
                          <label className="text-sm font-medium text-neutral-700 mb-1.5 block">
                            Infant (Bayi)
                          </label>
                          <div className="flex items-center gap-2 h-10">
                            <Toggle
                              id={`toggle_infant_${pax.id}`}
                              name={`toggle_infant_${pax.id}`}
                              checked={pax.is_infant}
                              onChange={(e) => handlePaxChange(pax.id, "is_infant", e.target.checked)}
                            />
                            <span className="text-xs text-neutral-600">
                              {pax.is_infant ? "Ya" : "Tidak"}
                            </span>
                          </div>
                        </>
                      )}
                    </div>

                    {/* Tipe Kamar */}
                    <div className="md:col-span-3">
                      <CustomDropdown
                        label="Tipe Kamar"
                        value={pax.is_infant ? "" : pax.room_type || ""}
                        onChange={(val) => handlePaxChange(pax.id, "room_type", val)}
                        placeholder={pax.is_infant ? "Tanpa kamar (Infant)" : "-- Pilih Tipe Kamar --"}
                        disabled={pax.is_infant}
                        options={[
                          { value: "Quad", label: "Quad (4 orang)" },
                          { value: "Triple", label: "Triple (3 orang)" },
                          { value: "Double", label: "Double (2 orang)" }
                        ]}
                        error={roomError}
                      />
                    </div>

                    {/* Live Harga Pax */}
                    <div className="md:col-span-2 flex flex-col justify-start">
                      <label className="text-sm font-medium text-neutral-700 mb-1.5 block">
                        Harga Pax
                      </label>
                      <div className="h-10 flex items-center">
                        <span className="font-heading font-semibold text-sm text-neutral-900">
                          {paxPrice !== null ? formatRupiah(paxPrice) : "-"}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          <div className="mt-4 flex justify-center">
            <Button
              type="button"
              variant="secondary"
              onClick={handleAddPax}
              className="w-full sm:w-auto flex items-center justify-center gap-2"
            >
              <Plus className="w-4 h-4" />
              Tambah Pax Lagi
            </Button>
          </div>
        </MetaBox>

        {/* METABOX 3: Kontak Utama (PIC) */}
        <MetaBox title="Kontak Utama (PIC Booking)">
          <div className="space-y-3">
            <CustomDropdown
              label="Pilih Kontak Utama (PIC)"
              value={picJamaahId}
              onChange={(val) => {
                const actualVal = (val && typeof val === 'object' && 'target' in val) ? val.target.value : val;
                setPicJamaahId(actualVal);
                setClientErrors((prev) => {
                  const next = { ...prev };
                  delete next.pic_jamaah_id;
                  return next;
                });
              }}
              placeholder={
                picOptions.length === 0
                  ? "-- Isi nama jamaah reguler di atas dulu --"
                  : "-- Pilih Kontak Utama --"
              }
              options={picOptions}
              disabled={picOptions.length === 0}
              error={clientErrors.pic_jamaah_id}
              required
            />
            <p className="text-xs text-neutral-500">
              Pilih jamaah reguler dari daftar pax di atas yang bertanggung jawab sebagai PIC invoice & komunikasi.
            </p>
          </div>
        </MetaBox>

        {/* METABOX 4: Ringkasan & Total Harga */}
        <MetaBox title="Ringkasan & Total Harga">
          <div className="space-y-4">
            {selectedSchedule ? (
              <>
                {/* Status Kuota Kursi */}
                <div
                  className={`p-3.5 rounded-lg border flex items-start gap-3 ${
                    isOverQuota
                      ? "bg-danger-50 text-danger-800 border-danger-200"
                      : "bg-neutral-50 text-neutral-700 border-neutral-200"
                  }`}
                >
                  {isOverQuota ? (
                    <AlertCircle className="w-5 h-5 text-danger-600 shrink-0 mt-0.5" />
                  ) : (
                    <Info className="w-5 h-5 text-primary-600 shrink-0 mt-0.5" />
                  )}
                  <div className="text-sm">
                    {isOverQuota ? (
                      <div>
                        <span className="font-semibold">Melebihi Sisa Kuota Kursi!</span>
                        <p className="mt-0.5">
                          Booking ini membutuhkan <strong>{regularPaxCount} kursi reguler</strong>, namun sisa kuota kursi pada paket hanya <strong>{seatSisa} kursi</strong>.
                        </p>
                      </div>
                    ) : (
                      <div>
                        <span className="font-semibold">Ketersediaan Kuota Aman</span>
                        <p className="mt-0.5 text-neutral-600">
                          <strong>{regularPaxCount}</strong> dari <strong>{seatSisa}</strong> kursi tersisa akan terpakai.
                          {infantPaxCount > 0 && ` (${infantPaxCount} infant tidak mengurangi kuota kursi)`}
                        </p>
                      </div>
                    )}
                  </div>
                </div>

                {/* Total Harga Bar */}
                <div className="flex flex-col sm:flex-row items-center justify-between p-4 bg-primary-50 border border-primary-200 rounded-lg">
                  <div>
                    <span className="text-xs uppercase tracking-wider font-semibold text-primary-800">
                      Estimasi Total Harga ({paxList.length} Pax)
                    </span>
                    <p className="text-xs text-primary-600">
                      {regularPaxCount} Reguler + {infantPaxCount} Infant
                    </p>
                  </div>
                  <div className="mt-2 sm:mt-0 font-heading font-bold text-2xl md:text-3xl text-primary-900">
                    {formatRupiah(totalHarga)}
                  </div>
                </div>
              </>
            ) : (
              <div className="p-3.5 rounded-lg bg-neutral-50 border border-neutral-200 text-neutral-500 text-sm">
                Pilih paket di atas untuk melihat ketersediaan kuota kursi dan kalkulasi harga.
              </div>
            )}
          </div>
        </MetaBox>

        {/* Action Buttons */}
        <div className="flex items-center justify-end gap-3 pt-4 border-t border-neutral-200">
          <Button type="button" variant="ghost" onClick={() => navigate("/bookings")}>
            Batal
          </Button>
          <Button
            type="submit"
            variant="primary"
            disabled={isSubmitDisabled}
          >
            {submitting ? "Memproses..." : "Buat Booking"}
          </Button>
        </div>
      </form>

      {/* Modal Konfirmasi Cascade Delete */}
      <Modal
        isOpen={Boolean(cascadeDeleteTarget)}
        onClose={() => setCascadeDeleteTarget(null)}
        title="Konfirmasi Hapus Pax"
        footer={
          <div className="flex items-center justify-end gap-3">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setCascadeDeleteTarget(null)}
            >
              Batal
            </Button>
            <Button
              type="button"
              variant="danger"
              onClick={handleConfirmCascadeDelete}
            >
              Hapus Semua
            </Button>
          </div>
        }
      >
        <p className="text-sm text-neutral-700 font-body">
          Menghapus <strong>{cascadeDeleteTarget ? (jamaahMap[cascadeDeleteTarget.targetPax.jamaah_id]?.nama_lengkap || `Pax #${paxList.findIndex((p) => p.id === cascadeDeleteTarget.targetPax.id) + 1}`) : ''}</strong> akan menghapus juga <strong>{cascadeDeleteTarget?.infantCount} pax infant</strong> lain yang menyertainya, karena infant tidak bisa terdaftar sendiri tanpa pax reguler penyerta. Lanjutkan?
        </p>
      </Modal>
    </div>
  );
};

export default BookingFormPage;
