import React, { useState, useEffect } from 'react';
import { Plane, MapPin, Pencil, Trash2 } from 'lucide-react';
import PageHeader from 'shared/ui/PageHeader';
import DataTable from 'shared/ui/DataTable';
import Button from 'shared/ui/Button';
import Modal from 'shared/ui/Modal';
import Input from 'shared/ui/Input';
import FileInput from 'shared/ui/FileInput';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import ActionMenu from 'shared/ui/ActionMenu';
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

  if (!url || error) {
    return (
      <div className="w-10 h-10 shrink-0 bg-neutral-100 rounded-lg border border-neutral-200 flex items-center justify-center text-neutral-400">
        <Plane className="w-4 h-4" strokeWidth={1.75} />
      </div>
    );
  }

  return (
    <img
      src={url.startsWith('/') ? `${import.meta.env.VITE_API_BASE_URL}${url}` : url}
      alt={`${name} logo`}
      className="w-10 h-10 shrink-0 object-contain rounded-lg border border-neutral-200 bg-white p-1"
      onError={() => setError(true)}
    />
  );
};

const CodeText = ({ code }) => (
  code
    ? <span className="font-mono text-[13px] font-semibold tracking-wider text-neutral-800">{code}</span>
    : <span className="text-xs text-neutral-400 font-body">-</span>
);

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

  // ─── Tampilan ───────────────────────────────────────────────────────────────
  const tabs = [
    { id: 'airlines', label: 'Maskapai', count: airlines.length },
    { id: 'airports', label: 'Bandara', count: airports.length },
  ];

  const tabSwitcher = (
    <div className="flex flex-wrap gap-1.5">
      {tabs.map(tab => {
        const isActive = activeTab === tab.id;
        return (
          <button
            key={tab.id}
            type="button"
            onClick={() => setActiveTab(tab.id)}
            className={`inline-flex items-center gap-2 px-3 py-1.5 text-xs font-medium font-body rounded-lg border transition-colors ${
              isActive
                ? 'bg-primary-500 border-primary-500 text-white'
                : 'bg-white border-neutral-200 text-neutral-600 hover:bg-neutral-50'
            }`}
          >
            <span>{tab.label}</span>
            <span className={`min-w-[20px] px-1.5 py-0.5 rounded-full text-[11px] leading-none text-center ${
              isActive ? 'bg-white/25 text-white' : 'bg-neutral-100 text-neutral-600'
            }`}>
              {tab.count}
            </span>
          </button>
        );
      })}
    </div>
  );

  const airlineColumns = [
    { header: 'Maskapai', key: 'name', sortable: true },
    { header: 'Kode IATA', key: 'code', sortable: true },
    { header: 'Aksi', key: 'aksi' },
  ];

  const renderAirlineCell = (row, key) => {
    if (key === 'name') {
      return (
        <div className="flex items-center gap-3 min-w-0">
          <LogoCell url={row.logo_url} name={row.name} />
          <button
            type="button"
            onClick={() => handleOpenAirlineModal(row)}
            className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors"
          >
            {row.name}
          </button>
        </div>
      );
    }
    if (key === 'code') return <CodeText code={row.code} />;
    if (key === 'aksi') {
      const menuItems = [
        { label: 'Edit', icon: Pencil, tooltip: 'Edit Maskapai', onClick: () => handleOpenAirlineModal(row) },
        { label: 'Hapus', icon: Trash2, danger: true, tooltip: 'Hapus Maskapai', onClick: () => setDeleteConfirmAirlineId(row.id) },
      ];
      return <ActionMenu items={menuItems} align="right" />;
    }
    return row[key];
  };

  const airportColumns = [
    { header: 'Bandara', key: 'name', sortable: true },
    { header: 'Kode', key: 'code', sortable: true },
    { header: 'Kota', key: 'city', sortable: true },
    { header: 'Aksi', key: 'aksi' },
  ];

  const renderAirportCell = (row, key) => {
    if (key === 'name') {
      return (
        <button
          type="button"
          onClick={() => handleOpenAirportModal(row)}
          className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left leading-snug transition-colors"
        >
          {row.name}
        </button>
      );
    }
    if (key === 'code') return <CodeText code={row.code} />;
    if (key === 'city') {
      return (
        <span className="inline-flex items-center gap-1.5 text-[13px] font-body text-neutral-700">
          <MapPin className="w-3.5 h-3.5 text-neutral-400" strokeWidth={1.75} />
          {row.city}
        </span>
      );
    }
    if (key === 'aksi') {
      const menuItems = [
        { label: 'Edit', icon: Pencil, tooltip: 'Edit Bandara', onClick: () => handleOpenAirportModal(row) },
        { label: 'Hapus', icon: Trash2, danger: true, tooltip: 'Hapus Bandara', onClick: () => setDeleteConfirmAirportId(row.id) },
      ];
      return <ActionMenu items={menuItems} align="right" />;
    }
    return row[key];
  };

  const isLoadingTab = activeTab === 'airlines' ? isAirlinesLoading : isAirportsLoading;

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

      {airlineErrorMessage && (
        <Alert variant="error" onClose={() => setAirlineErrorMessage(null)}>{airlineErrorMessage}</Alert>
      )}
      {airportErrorMessage && (
        <Alert variant="error" onClose={() => setAirportErrorMessage(null)}>{airportErrorMessage}</Alert>
      )}

      {isLoadingTab ? (
        <div className="flex justify-center p-8 bg-white rounded-2xl border border-neutral-200 shadow-card"><LoadingSpinner /></div>
      ) : activeTab === 'airlines' ? (
        <DataTable
          key="airlines-table"
          columns={airlineColumns}
          data={airlines}
          itemsPerPage={10}
          searchPlaceholder="Cari maskapai..."
          emptyMessage='Belum ada maskapai. Klik "+ Tambah Maskapai" untuk menambahkan.'
          toolbarActions={tabSwitcher}
          renderCell={renderAirlineCell}
        />
      ) : (
        <DataTable
          key="airports-table"
          columns={airportColumns}
          data={airports}
          itemsPerPage={10}
          searchPlaceholder="Cari bandara (nama, kode, kota)..."
          emptyMessage='Belum ada bandara. Klik "+ Tambah Bandara" untuk menambahkan.'
          toolbarActions={tabSwitcher}
          renderCell={renderAirportCell}
        />
      )}

      {/* ─── Modal Form Maskapai ──────────────────────────────────────────── */}
      <Modal
        isOpen={isAirlineModalOpen}
        onClose={handleCloseAirlineModal}
        title={editingAirline ? "Edit Maskapai" : "Tambah Maskapai"}
        size="md"
      >
        <form onSubmit={handleAirlineSubmit} className="space-y-4 font-body">
          <Input
            label="Nama"
            name="name"
            value={airlineFormData.name}
            onChange={(e) => setAirlineFormData(prev => ({ ...prev, name: e.target.value }))}
            required
            placeholder="Contoh: Saudia Airlines"
            className="!mb-0"
          />

          <Input
            label="Kode IATA"
            name="code"
            value={airlineFormData.code}
            onChange={(e) => {
              const val = e.target.value.toUpperCase().slice(0, 2);
              setAirlineFormData(prev => ({ ...prev, code: val }));
            }}
            maxLength={2}
            placeholder="Contoh: SV"
            className="!mb-0"
          />

          <FileInput
            label="Logo"
            value={airlineFormData.logo_url}
            previewUrl={logoPreview}
            onChange={handleLogoFileChange}
            onRemove={() => {
              setLogoPreview(null);
              setAirlineFormData(prev => ({ ...prev, logo_url: '' }));
            }}
            isUploading={isUploadingLogo}
            uploadingText="Mengupload logo maskapai..."
            placeholder="Pilih file..."
            className="!mb-0"
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
          <Input
            label="Nama"
            name="name"
            value={airportFormData.name}
            onChange={(e) => setAirportFormData(prev => ({ ...prev, name: e.target.value }))}
            required
            placeholder="Contoh: Soekarno-Hatta International Airport"
            className="!mb-0"
          />

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Input
              label="Kode IATA"
              name="code"
              value={airportFormData.code}
              onChange={(e) => {
                const val = e.target.value.toUpperCase().slice(0, 4);
                setAirportFormData(prev => ({ ...prev, code: val }));
              }}
              required
              maxLength={4}
              placeholder="Contoh: CGK"
              className="!mb-0"
            />

            <Input
              label="Kota"
              name="city"
              value={airportFormData.city}
              onChange={(e) => setAirportFormData(prev => ({ ...prev, city: e.target.value }))}
              required
              placeholder="Contoh: Jakarta"
              className="!mb-0"
            />
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
