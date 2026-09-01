import React, { useState, useEffect } from 'react';
import PageHeader from '../components/ui/PageHeader';
import DataTable from '../components/ui/DataTable';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import FormField from '../components/ui/FormField';
import Input from '../components/ui/Input';
import FileInput from '../components/ui/FileInput';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import { listAirlines, createAirline, updateAirline, deleteAirline } from '../api/airlines';
import { listAirports, createAirport, updateAirport, deleteAirport } from '../api/airports';
import { uploadMedia } from '../api/media';

const initialAirlineForm = {
  name: '',
  code: '',
  logo_url: ''
};

const initialAirportForm = {
  name: '',
  code: '',
  city: ''
};

const LogoCell = ({ url, name }) => {
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
      alt={`${name} logo`}
      className="w-8 h-8 object-contain rounded bg-white"
      onError={() => setError(true)}
    />
  );
};

const AirlinesPage = () => {
  const [activeTab, setActiveTab] = useState('airlines'); // 'airlines' | 'airports'

  // ─── Airlines State ──────────────────────────────────────────────────────────
  const [airlines, setAirlines] = useState([]);
  const [isAirlinesLoading, setIsAirlinesLoading] = useState(true);
  const [airlineErrorMessage, setAirlineErrorMessage] = useState(null);
  
  const [isAirlineModalOpen, setIsAirlineModalOpen] = useState(false);
  const [editingAirline, setEditingAirline] = useState(null);
  const [airlineFormData, setAirlineFormData] = useState(initialAirlineForm);
  const [airlineFormErrors, setAirlineFormErrors] = useState(null);
  const [isUploadingLogo, setIsUploadingLogo] = useState(false);
  const [logoPreview, setLogoPreview] = useState(null);
  const [isSubmittingAirline, setIsSubmittingAirline] = useState(false);
  const [deleteConfirmAirlineId, setDeleteConfirmAirlineId] = useState(null);

  // ─── Airports State ──────────────────────────────────────────────────────────
  const [airports, setAirports] = useState([]);
  const [isAirportsLoading, setIsAirportsLoading] = useState(true);
  const [airportErrorMessage, setAirportErrorMessage] = useState(null);
  
  const [isAirportModalOpen, setIsAirportModalOpen] = useState(false);
  const [editingAirport, setEditingAirport] = useState(null);
  const [airportFormData, setAirportFormData] = useState(initialAirportForm);
  const [airportFormErrors, setAirportFormErrors] = useState(null);
  const [isSubmittingAirport, setIsSubmittingAirport] = useState(false);
  const [deleteConfirmAirportId, setDeleteConfirmAirportId] = useState(null);

  // ─── Fetch Data ─────────────────────────────────────────────────────────────
  const fetchAirlines = async () => {
    setIsAirlinesLoading(true);
    setAirlineErrorMessage(null);
    try {
      const data = await listAirlines();
      setAirlines(data || []);
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal memuat data maskapai.";
      setAirlineErrorMessage(msg);
    } finally {
      setIsAirlinesLoading(false);
    }
  };

  const fetchAirports = async () => {
    setIsAirportsLoading(true);
    setAirportErrorMessage(null);
    try {
      const data = await listAirports();
      setAirports(data || []);
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal memuat data bandara.";
      setAirportErrorMessage(msg);
    } finally {
      setIsAirportsLoading(false);
    }
  };

  useEffect(() => {
    fetchAirlines();
    fetchAirports();
  }, []);

  // ─── Airlines Modal & Actions ───────────────────────────────────────────────
  const handleOpenAirlineModal = (airline = null) => {
    if (airline) {
      setEditingAirline(airline);
      setAirlineFormData({
        name: airline.name,
        code: airline.code || '',
        logo_url: airline.logo_url || ''
      });
      setLogoPreview(null);
    } else {
      setEditingAirline(null);
      setAirlineFormData(initialAirlineForm);
      setLogoPreview(null);
    }
    setAirlineFormErrors(null);
    setIsAirlineModalOpen(true);
  };

  const handleCloseAirlineModal = () => {
    if (!isSubmittingAirline) {
      setIsAirlineModalOpen(false);
      setEditingAirline(null);
      setAirlineFormData(initialAirlineForm);
      setLogoPreview(null);
    }
  };

  const handleLogoFileChange = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    const objectUrl = URL.createObjectURL(file);
    setLogoPreview(objectUrl);
    setAirlineFormErrors(null);
    setIsUploadingLogo(true);

    try {
      const url = await uploadMedia(file, 'airline-logos');
      setAirlineFormData(prev => ({ ...prev, logo_url: url }));
    } catch (err) {
      console.error(err);
      setAirlineFormErrors('Gagal upload logo, coba lagi');
    } finally {
      setIsUploadingLogo(false);
    }
  };

  const handleAirlineSubmit = async (e) => {
    e.preventDefault();
    setAirlineFormErrors(null);
    setIsSubmittingAirline(true);

    try {
      const payload = {
        name: airlineFormData.name.trim(),
        code: airlineFormData.code.trim() ? airlineFormData.code.trim().toUpperCase() : null,
        logo_url: airlineFormData.logo_url === '' ? null : airlineFormData.logo_url
      };

      if (editingAirline) {
        await updateAirline(editingAirline.id, payload);
      } else {
        await createAirline(payload);
      }

      handleCloseAirlineModal();
      fetchAirlines();
    } catch (error) {
      const msg = error.response?.data?.error || "Terjadi kesalahan, coba lagi.";
      setAirlineFormErrors(msg);
    } finally {
      setIsSubmittingAirline(false);
    }
  };

  const handleDeleteAirline = async () => {
    try {
      await deleteAirline(deleteConfirmAirlineId);
      setDeleteConfirmAirlineId(null);
      fetchAirlines();
    } catch (error) {
      setDeleteConfirmAirlineId(null);
      if (error.response?.status === 409) {
        setAirlineErrorMessage(error.response?.data?.error || "Tidak bisa dihapus, masih dipakai oleh paket lain.");
      } else {
        setAirlineErrorMessage("Terjadi kesalahan saat menghapus data.");
      }
    }
  };

  const airlineToDelete = airlines.find(a => a.id === deleteConfirmAirlineId);

  // ─── Airports Modal & Actions ───────────────────────────────────────────────
  const handleOpenAirportModal = (airport = null) => {
    if (airport) {
      setEditingAirport(airport);
      setAirportFormData({
        name: airport.name,
        code: airport.code,
        city: airport.city
      });
    } else {
      setEditingAirport(null);
      setAirportFormData(initialAirportForm);
    }
    setAirportFormErrors(null);
    setIsAirportModalOpen(true);
  };

  const handleCloseAirportModal = () => {
    if (!isSubmittingAirport) {
      setIsAirportModalOpen(false);
      setEditingAirport(null);
      setAirportFormData(initialAirportForm);
    }
  };

  const handleAirportSubmit = async (e) => {
    e.preventDefault();
    setAirportFormErrors(null);
    setIsSubmittingAirport(true);

    try {
      const payload = {
        name: airportFormData.name.trim(),
        code: airportFormData.code.trim().toUpperCase(),
        city: airportFormData.city.trim()
      };

      if (editingAirport) {
        await updateAirport(editingAirport.id, payload);
      } else {
        await createAirport(payload);
      }

      handleCloseAirportModal();
      fetchAirports();
    } catch (error) {
      const msg = error.response?.data?.error || "Terjadi kesalahan, coba lagi.";
      setAirportFormErrors(msg);
    } finally {
      setIsSubmittingAirport(false);
    }
  };

  const handleDeleteAirport = async () => {
    try {
      await deleteAirport(deleteConfirmAirportId);
      setDeleteConfirmAirportId(null);
      fetchAirports();
    } catch (error) {
      setDeleteConfirmAirportId(null);
      if (error.response?.status === 409) {
        setAirportErrorMessage(error.response?.data?.error || "Tidak bisa dihapus, masih dipakai oleh data lain.");
      } else {
        setAirportErrorMessage("Terjadi kesalahan saat menghapus bandara.");
      }
    }
  };

  const airportToDelete = airports.find(a => a.id === deleteConfirmAirportId);

  return (
    <div className="space-y-6">
      <PageHeader 
        title="Maskapai & Bandara" 
        actionLabel={activeTab === 'airlines' ? "+ Tambah Maskapai" : "+ Tambah Bandara"} 
        onAction={() => {
          if (activeTab === 'airlines') {
            handleOpenAirlineModal();
          } else {
            handleOpenAirportModal();
          }
        }}
      />

      {/* Tabs */}
      <div className="flex items-center gap-2 border-b border-neutral-200 pb-3">
        <button
          type="button"
          onClick={() => setActiveTab('airlines')}
          className={`inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full text-xs font-semibold font-heading transition-all ${
            activeTab === 'airlines'
              ? 'bg-sidebar-bg text-white shadow-2xs font-bold'
              : 'bg-white text-neutral-600 hover:bg-neutral-50 border border-neutral-200/90'
          }`}
        >
          <span>Maskapai</span>
          <span className={`px-2 py-0.5 rounded-full text-[11px] ${activeTab === 'airlines' ? 'bg-white/20 text-white' : 'bg-neutral-100 text-neutral-600'}`}>
            {airlines.length}
          </span>
        </button>

        <button
          type="button"
          onClick={() => setActiveTab('airports')}
          className={`inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full text-xs font-semibold font-heading transition-all ${
            activeTab === 'airports'
              ? 'bg-sidebar-bg text-white shadow-2xs font-bold'
              : 'bg-white text-neutral-600 hover:bg-neutral-50 border border-neutral-200/90'
          }`}
        >
          <span>Bandara</span>
          <span className={`px-2 py-0.5 rounded-full text-[11px] ${activeTab === 'airports' ? 'bg-white/20 text-white' : 'bg-neutral-100 text-neutral-600'}`}>
            {airports.length}
          </span>
        </button>
      </div>

      {/* ─── Tab Maskapai ─────────────────────────────────────────────────── */}
      {activeTab === 'airlines' && (
        <div className="space-y-4">
          {airlineErrorMessage && (
            <Alert variant="error">{airlineErrorMessage}</Alert>
          )}

          {isAirlinesLoading ? (
            <div className="flex justify-center p-8 bg-white rounded-lg border border-neutral-200 shadow-sm">
              <LoadingSpinner />
            </div>
          ) : (
            <DataTable
              key="airlines-table"
              columns={[
                { header: 'Logo', key: 'logo_url' },
                { header: 'Nama Maskapai', key: 'name', sortable: true },
                { header: 'Kode', key: 'code', align: 'center', sortable: true },
                { header: 'Aksi', key: 'aksi' },
              ]}
              data={airlines}
              itemsPerPage={10}
              searchPlaceholder="Cari maskapai..."
              emptyMessage='Belum ada maskapai. Klik "+ Tambah Maskapai" untuk menambahkan.'
              renderCell={(row, key) => {
                if (key === 'logo_url') {
                  return <LogoCell url={row.logo_url} name={row.name} />;
                }
                if (key === 'code') {
                  return (
                    <div className="flex items-center justify-center">
                      {row.code ? (
                        <span className="font-mono text-xs font-bold text-neutral-800 bg-neutral-100 px-2.5 py-0.5 rounded-lg border border-neutral-200/90 tracking-wider">
                          {row.code}
                        </span>
                      ) : (
                        <span className="text-neutral-400 text-xs">-</span>
                      )}
                    </div>
                  );
                }
                if (key === 'aksi') return (
                  <div className="flex gap-3">
                    <button 
                      onClick={() => handleOpenAirlineModal(row)} 
                      title="Edit"
                      className="text-neutral-400 hover:text-neutral-700 transition-colors"
                    >
                      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                      </svg>
                    </button>
                    <button 
                      onClick={() => setDeleteConfirmAirlineId(row.id)} 
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
        </div>
      )}

      {/* ─── Tab Bandara ──────────────────────────────────────────────────── */}
      {activeTab === 'airports' && (
        <div className="space-y-4">
          {airportErrorMessage && (
            <Alert variant="error">{airportErrorMessage}</Alert>
          )}

          {isAirportsLoading ? (
            <div className="flex justify-center p-8 bg-white rounded-lg border border-neutral-200 shadow-sm">
              <LoadingSpinner />
            </div>
          ) : (
            <DataTable
              key="airports-table"
              columns={[
                { header: 'Nama Bandara', key: 'name', sortable: true },
                { 
                  header: 'Kode', 
                  key: 'code', 
                  align: 'center', 
                  sortable: true 
                },
                { header: 'Kota', key: 'city', sortable: true },
                { header: 'Aksi', key: 'aksi' },
              ]}
              data={airports}
              itemsPerPage={10}
              searchPlaceholder="Cari bandara (nama, kode, kota)..."
              emptyMessage='Belum ada bandara. Klik "+ Tambah Bandara" untuk menambahkan.'
              renderCell={(row, key) => {
                if (key === 'code') {
                  return (
                    <div className="flex items-center justify-center">
                      <span className="font-mono text-xs font-bold text-neutral-800 bg-neutral-100 px-2.5 py-0.5 rounded-lg border border-neutral-200/90 tracking-wider">
                        {row.code}
                      </span>
                    </div>
                  );
                }
                if (key === 'aksi') return (
                  <div className="flex gap-3">
                    <button 
                      onClick={() => handleOpenAirportModal(row)} 
                      title="Edit"
                      className="text-neutral-400 hover:text-neutral-700 transition-colors"
                    >
                      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                      </svg>
                    </button>
                    <button 
                      onClick={() => setDeleteConfirmAirportId(row.id)} 
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
        </div>
      )}

      {/* ─── Modal Form Maskapai ──────────────────────────────────────────── */}
      <Modal 
        isOpen={isAirlineModalOpen} 
        onClose={handleCloseAirlineModal}
        title={editingAirline ? "Edit Maskapai" : "Tambah Maskapai"}
        size="md"
      >
        <form onSubmit={handleAirlineSubmit} className="space-y-4 font-body">
          <FormField label="Nama Maskapai" required>
            <Input 
              name="name"
              value={airlineFormData.name}
              onChange={(e) => setAirlineFormData(prev => ({ ...prev, name: e.target.value }))}
              required
              placeholder="Contoh: Saudia Airlines"
            />
          </FormField>

          <FormField label="Kode IATA (Opsional, 2 Huruf)">
            <Input 
              name="code"
              value={airlineFormData.code}
              onChange={(e) => {
                const val = e.target.value.toUpperCase().slice(0, 2);
                setAirlineFormData(prev => ({ ...prev, code: val }));
              }}
              maxLength={2}
              placeholder="Contoh: SV"
              className="uppercase font-mono font-semibold"
            />
          </FormField>

          <FileInput
            label="Logo Maskapai (Opsional)"
            value={airlineFormData.logo_url}
            previewUrl={logoPreview}
            onChange={handleLogoFileChange}
            onRemove={() => {
              setLogoPreview(null);
              setAirlineFormData(prev => ({ ...prev, logo_url: '' }));
            }}
            isUploading={isUploadingLogo}
            uploadingText="Mengupload logo maskapai..."
            placeholder="Pilih file logo maskapai (PNG, SVG, JPG)..."
            helperText="Format transparan PNG atau SVG direkomendasikan"
          />

          {airlineFormErrors && (
            <Alert variant="error">{airlineFormErrors}</Alert>
          )}

          <div className="pt-4 flex justify-end space-x-3 border-t border-neutral-200">
            <Button type="button" variant="ghost" onClick={handleCloseAirlineModal} disabled={isSubmittingAirline}>
              Batal
            </Button>
            <Button type="submit" variant="primary" isLoading={isSubmittingAirline}>
              {editingAirline ? "Update" : "Simpan"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* ─── Modal Form Bandara ───────────────────────────────────────────── */}
      <Modal 
        isOpen={isAirportModalOpen} 
        onClose={handleCloseAirportModal}
        title={editingAirport ? "Edit Bandara" : "Tambah Bandara"}
        size="md"
      >
        <form onSubmit={handleAirportSubmit} className="space-y-4 font-body">
          <FormField label="Nama Bandara" required>
            <Input 
              name="name"
              value={airportFormData.name}
              onChange={(e) => setAirportFormData(prev => ({ ...prev, name: e.target.value }))}
              required
              placeholder="Contoh: Soekarno-Hatta International Airport"
            />
          </FormField>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <FormField label="Kode Bandara (IATA)" required>
              <Input 
                name="code"
                value={airportFormData.code}
                onChange={(e) => {
                  const val = e.target.value.toUpperCase().slice(0, 4);
                  setAirportFormData(prev => ({ ...prev, code: val }));
                }}
                required
                maxLength={4}
                placeholder="Contoh: CGK"
                className="uppercase font-mono font-semibold"
              />
            </FormField>

            <FormField label="Kota Bandara" required>
              <Input 
                name="city"
                value={airportFormData.city}
                onChange={(e) => setAirportFormData(prev => ({ ...prev, city: e.target.value }))}
                required
                placeholder="Contoh: Tangerang (Jakarta)"
              />
            </FormField>
          </div>

          {airportFormErrors && (
            <Alert variant="error">{airportFormErrors}</Alert>
          )}

          <div className="pt-4 flex justify-end space-x-3 border-t border-neutral-200">
            <Button type="button" variant="ghost" onClick={handleCloseAirportModal} disabled={isSubmittingAirport}>
              Batal
            </Button>
            <Button type="submit" variant="primary" isLoading={isSubmittingAirport}>
              {editingAirport ? "Update" : "Simpan"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* ─── Delete Confirmation Modal (Maskapai) ─────────────────────────── */}
      <Modal
        isOpen={deleteConfirmAirlineId !== null}
        onClose={() => setDeleteConfirmAirlineId(null)}
        title="Hapus Maskapai?"
        size="sm"
        footer={
          <>
            <Button variant="ghost" onClick={() => setDeleteConfirmAirlineId(null)}>Batal</Button>
            <Button variant="danger" onClick={handleDeleteAirline}>Hapus</Button>
          </>
        }
      >
        <p className="text-neutral-600 font-body">
          Yakin ingin menghapus {airlineToDelete?.name}? Tindakan ini tidak bisa dibatalkan.
        </p>
      </Modal>

      {/* ─── Delete Confirmation Modal (Bandara) ──────────────────────────── */}
      <Modal
        isOpen={deleteConfirmAirportId !== null}
        onClose={() => setDeleteConfirmAirportId(null)}
        title="Hapus Bandara?"
        size="sm"
        footer={
          <>
            <Button variant="ghost" onClick={() => setDeleteConfirmAirportId(null)}>Batal</Button>
            <Button variant="danger" onClick={handleDeleteAirport}>Hapus</Button>
          </>
        }
      >
        <p className="text-neutral-600 font-body">
          Yakin ingin menghapus bandara <strong>{airportToDelete?.name} ({airportToDelete?.code})</strong>? Tindakan ini tidak bisa dibatalkan.
        </p>
      </Modal>
    </div>
  );
};

export default AirlinesPage;
