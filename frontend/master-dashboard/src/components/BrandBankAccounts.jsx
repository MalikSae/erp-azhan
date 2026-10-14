import React, { useEffect, useState } from 'react';
import { Landmark, Pencil, Trash2, Plus } from 'lucide-react';
import Button from 'shared/ui/Button';
import Input from 'shared/ui/Input';
import Modal from 'shared/ui/Modal';
import Alert from 'shared/ui/Alert';
import FormField from 'shared/ui/FormField';
import FileInput from 'shared/ui/FileInput';
import Toggle from 'shared/ui/Toggle';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import { listBankAccounts, createBankAccount, updateBankAccount, deleteBankAccount } from '../api/bankAccounts';
import { uploadMedia } from '../api/media';

const emptyForm = {
  bank_name: '',
  logo_url: '',
  account_number: '',
  account_holder: '',
  instructions: '',
  is_active: true,
  sort_order: 0,
};

const logoSrcOf = (url) => {
  if (!url) return null;
  if (url.startsWith('http')) return url;
  return `${import.meta.env.VITE_API_BASE_URL || 'http://localhost:9090'}${url.startsWith('/') ? '' : '/'}${url}`;
};

const BankLogo = ({ url, name }) => {
  const [failed, setFailed] = useState(false);
  const src = logoSrcOf(url);
  if (!src || failed) {
    return (
      <div className="w-10 h-10 shrink-0 rounded-lg border border-neutral-200 bg-neutral-100 flex items-center justify-center text-neutral-400">
        <Landmark className="w-4 h-4" strokeWidth={1.75} />
      </div>
    );
  }
  return (
    <img
      src={src}
      alt={name}
      className="w-10 h-10 shrink-0 object-contain rounded-lg border border-neutral-200 bg-white p-1"
      onError={() => setFailed(true)}
    />
  );
};

// Rekening bank milik satu brand. Disimpan langsung per rekening (terpisah dari tombol simpan profil brand).
const BrandBankAccounts = ({ brandId }) => {
  const [items, setItems] = useState([]);
  const [isLoading, setIsLoading] = useState(Boolean(brandId));
  const [error, setError] = useState('');
  const [open, setOpen] = useState(false);
  const [editingId, setEditingId] = useState(null);
  const [form, setForm] = useState(emptyForm);
  const [formError, setFormError] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [isUploadingLogo, setIsUploadingLogo] = useState(false);
  const [logoPreview, setLogoPreview] = useState(null);
  const [deleteTarget, setDeleteTarget] = useState(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const load = () =>
    listBankAccounts()
      .then((all) => setItems((all || []).filter((a) => String(a.brand_id) === String(brandId))))
      .catch(() => setError('Data rekening gagal dimuat.'))
      .finally(() => setIsLoading(false));

  useEffect(() => {
    if (brandId) load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [brandId]);

  if (!brandId) {
    return (
      <p className="text-sm text-neutral-500 font-body">
        Simpan profil brand terlebih dahulu, lalu rekening bank dapat ditambahkan di sini.
      </p>
    );
  }

  const openAdd = () => {
    setEditingId(null);
    setForm({ ...emptyForm, sort_order: items.length });
    setLogoPreview(null);
    setFormError('');
    setOpen(true);
  };

  const openEdit = (a) => {
    setEditingId(a.id);
    setForm({
      bank_name: a.bank_name || '',
      logo_url: a.logo_url || '',
      account_number: a.account_number || '',
      account_holder: a.account_holder || '',
      instructions: a.instructions || '',
      is_active: Boolean(a.is_active),
      sort_order: a.sort_order || 0,
    });
    setLogoPreview(null);
    setFormError('');
    setOpen(true);
  };

  const handleLogoChange = async (e) => {
    const file = e.target.files[0];
    if (!file) return;
    setLogoPreview(URL.createObjectURL(file));
    setIsUploadingLogo(true);
    setFormError('');
    try {
      const url = await uploadMedia(file, 'bank-logos');
      setForm((prev) => ({ ...prev, logo_url: url }));
    } catch (err) {
      setFormError(err.response?.data?.error || 'Gagal mengupload logo bank.');
      setLogoPreview(null);
    } finally {
      setIsUploadingLogo(false);
    }
  };

  const save = async () => {
    if (!form.bank_name.trim() || !form.account_number.trim() || !form.account_holder.trim()) {
      setFormError('Nama bank, nomor rekening, dan atas nama wajib diisi.');
      return;
    }
    setIsSaving(true);
    setFormError('');
    try {
      const payload = {
        ...form,
        brand_id: Number(brandId),
        sort_order: Number(form.sort_order) || 0,
        logo_url: form.logo_url || null,
        instructions: form.instructions ? form.instructions.trim() : null,
      };
      if (editingId) await updateBankAccount(editingId, payload);
      else await createBankAccount(payload);
      setOpen(false);
      load();
    } catch (err) {
      setFormError(err.response?.data?.error || 'Rekening gagal disimpan.');
    } finally {
      setIsSaving(false);
    }
  };

  const confirmDelete = async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await deleteBankAccount(deleteTarget.id);
      setDeleteTarget(null);
      load();
    } catch (err) {
      setDeleteTarget(null);
      setError(err.response?.data?.error || 'Rekening gagal dihapus.');
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <div className="space-y-4">
      {error && <Alert variant="error" onClose={() => setError('')}>{error}</Alert>}

      <div className="flex items-center justify-between gap-3">
        <p className="text-xs text-neutral-500 font-body">
          Rekening yang aktif tampil di formulir booking dan portal jamaah brand ini.
        </p>
        <Button type="button" variant="secondary" onClick={openAdd} className="shrink-0 whitespace-nowrap">
          <Plus size={14} strokeWidth={2} /> Tambah Rekening
        </Button>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-8"><LoadingSpinner /></div>
      ) : items.length === 0 ? (
        <div className="rounded-lg border border-dashed border-neutral-300 px-4 py-8 text-center text-sm text-neutral-500 font-body">
          Belum ada rekening bank untuk brand ini.
        </div>
      ) : (
        <ul className="divide-y divide-neutral-200 rounded-lg border border-neutral-200">
          {items.map((a) => (
            <li key={a.id} className="flex items-center gap-3 px-4 py-3">
              <BankLogo url={a.logo_url} name={a.bank_name} />
              <div className="min-w-0 flex-1">
                <div className="font-heading text-sm font-semibold text-neutral-900 truncate">{a.bank_name}</div>
                <div className="text-xs font-body text-neutral-500 truncate">
                  <span className="font-mono font-semibold tracking-wider text-neutral-700">{a.account_number}</span>
                  {' · a.n. '}{a.account_holder}
                </div>
              </div>
              <span className={`inline-flex items-center gap-1.5 text-[12px] font-body whitespace-nowrap ${a.is_active ? 'text-success-700' : 'text-neutral-500'}`}>
                <span className={`w-2 h-2 rounded-full ${a.is_active ? 'bg-success-500' : 'bg-neutral-300'}`} />
                {a.is_active ? 'Aktif' : 'Nonaktif'}
              </span>
              <button
                type="button"
                onClick={() => openEdit(a)}
                aria-label="Edit rekening"
                title="Edit rekening"
                className="p-1.5 rounded-md text-neutral-500 hover:text-neutral-900 hover:bg-neutral-100 cursor-pointer"
              >
                <Pencil size={15} strokeWidth={1.75} />
              </button>
              <button
                type="button"
                onClick={() => setDeleteTarget(a)}
                aria-label="Hapus rekening"
                title="Hapus rekening"
                className="p-1.5 rounded-md text-neutral-500 hover:text-danger-600 hover:bg-danger-50 cursor-pointer"
              >
                <Trash2 size={15} strokeWidth={1.75} />
              </button>
            </li>
          ))}
        </ul>
      )}

      <Modal
        isOpen={open}
        onClose={() => !isSaving && setOpen(false)}
        title={editingId ? 'Edit Rekening Bank' : 'Tambah Rekening Bank'}
        footer={
          <>
            <Button type="button" variant="ghost" onClick={() => setOpen(false)} disabled={isSaving}>Batal</Button>
            <Button type="button" onClick={save} isLoading={isSaving} disabled={isUploadingLogo}>Simpan</Button>
          </>
        }
      >
        <div className="space-y-4">
          {formError && <Alert variant="error">{formError}</Alert>}

          <Input
            label="Nama Bank"
            placeholder="Contoh: Bank Central Asia (BCA)"
            value={form.bank_name}
            onChange={(e) => setForm({ ...form, bank_name: e.target.value })}
            required
          />

          <FileInput
            label="Logo Bank"
            accept="image/png,image/jpeg,image/webp,image/svg+xml"
            value={form.logo_url}
            previewUrl={logoPreview}
            onChange={handleLogoChange}
            onRemove={() => {
              setLogoPreview(null);
              setForm((prev) => ({ ...prev, logo_url: '' }));
            }}
            isUploading={isUploadingLogo}
            uploadingText="Mengunggah logo bank..."
            placeholder="Pilih file logo bank (PNG, JPG, SVG)..."
          />

          <Input
            label="Nomor Rekening"
            placeholder="Contoh: 1234567890"
            value={form.account_number}
            onChange={(e) => setForm({ ...form, account_number: e.target.value.replace(/\D/g, '') })}
            required
          />

          <Input
            label="Atas Nama (Pemilik Rekening)"
            placeholder="Contoh: PT Azhan Tour & Travel"
            value={form.account_holder}
            onChange={(e) => setForm({ ...form, account_holder: e.target.value })}
            required
          />

          <Input
            label="Instruksi Pembayaran (Opsional)"
            placeholder="Contoh: Tambahkan kode unik saat transfer"
            value={form.instructions}
            onChange={(e) => setForm({ ...form, instructions: e.target.value })}
          />

          <FormField label="Status">
            <div className="flex items-center gap-3 pt-1">
              <Toggle
                id="brand_bank_is_active"
                name="is_active"
                checked={form.is_active}
                onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
              />
              <span className="text-sm font-body text-neutral-700">Aktif dan tampil di formulir booking / portal jamaah</span>
            </div>
          </FormField>
        </div>
      </Modal>

      <Modal
        isOpen={deleteTarget !== null}
        onClose={() => !isDeleting && setDeleteTarget(null)}
        title="Hapus Rekening?"
        size="sm"
        footer={
          <>
            <Button type="button" variant="ghost" onClick={() => setDeleteTarget(null)} disabled={isDeleting}>Batal</Button>
            <Button type="button" variant="danger" onClick={confirmDelete} isLoading={isDeleting}>Hapus</Button>
          </>
        }
      >
        <p className="text-neutral-600 font-body">
          Yakin ingin menghapus rekening <strong>{deleteTarget?.bank_name} - {deleteTarget?.account_number}</strong>? Tindakan ini tidak bisa dibatalkan.
        </p>
      </Modal>
    </div>
  );
};

export default BrandBankAccounts;
