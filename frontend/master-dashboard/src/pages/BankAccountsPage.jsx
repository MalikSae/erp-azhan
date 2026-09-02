import React, { useEffect, useState } from 'react';
import PageHeader from '../components/ui/PageHeader';
import Button from '../components/ui/Button';
import Input from '../components/ui/Input';
import Modal from '../components/ui/Modal';
import Alert from '../components/ui/Alert';
import FormField from '../components/ui/FormField';
import FileInput from '../components/ui/FileInput';
import { listBrands } from '../api/brands';
import { listBankAccounts, createBankAccount, updateBankAccount, deleteBankAccount } from '../api/bankAccounts';
import { uploadMedia } from '../api/media';

const empty = {
  brand_id: '',
  bank_name: '',
  logo_url: '',
  account_number: '',
  account_holder: '',
  instructions: '',
  is_active: true,
  sort_order: 0,
};

export default function BankAccountsPage() {
  const [items, setItems] = useState([]);
  const [brands, setBrands] = useState([]);
  const [form, setForm] = useState(empty);
  const [editing, setEditing] = useState(null);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState('');
  const [isUploadingLogo, setIsUploadingLogo] = useState(false);
  const [logoPreview, setLogoPreview] = useState(null);

  const load = () =>
    Promise.all([listBankAccounts(), listBrands()])
      .then(([a, b]) => {
        setItems(a || []);
        setBrands(b || []);
      })
      .catch(() => setError('Data rekening gagal dimuat.'));

  useEffect(() => {
    load();
  }, []);

  const handleOpenAdd = () => {
    setForm(empty);
    setEditing(null);
    setLogoPreview(null);
    setError('');
    setOpen(true);
  };

  const handleOpenEdit = (a) => {
    setEditing(a.id);
    setForm({
      ...a,
      brand_id: String(a.brand_id),
      logo_url: a.logo_url || '',
      instructions: a.instructions || '',
    });
    setLogoPreview(null);
    setError('');
    setOpen(true);
  };

  const handleLogoFileChange = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    const objectUrl = URL.createObjectURL(file);
    setLogoPreview(objectUrl);
    setIsUploadingLogo(true);
    setError('');

    try {
      const url = await uploadMedia(file, 'bank-logos');
      setForm((prev) => ({ ...prev, logo_url: url }));
    } catch (err) {
      setError(err.response?.data?.error || 'Gagal mengupload logo bank.');
      setLogoPreview(null);
    } finally {
      setIsUploadingLogo(false);
    }
  };

  const save = async (e) => {
    e.preventDefault();
    try {
      const payload = {
        ...form,
        brand_id: Number(form.brand_id),
        sort_order: Number(form.sort_order),
        logo_url: form.logo_url || null,
        instructions: form.instructions ? form.instructions.trim() : null,
      };
      editing ? await updateBankAccount(editing, payload) : await createBankAccount(payload);
      setOpen(false);
      setForm(empty);
      setEditing(null);
      setLogoPreview(null);
      load();
    } catch (err) {
      setError(err.response?.data?.error || 'Rekening gagal disimpan.');
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Rekening Bank"
        actionLabel="Tambah Rekening"
        onAction={handleOpenAdd}
      />

      {error && <Alert variant="error">{error}</Alert>}

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {items.map((a) => {
          const logoSrc = a.logo_url
            ? a.logo_url.startsWith('http')
              ? a.logo_url
              : `${import.meta.env.VITE_API_BASE_URL || 'http://localhost:9090'}${a.logo_url.startsWith('/') ? '' : '/'}${a.logo_url}`
            : null;

          return (
            <article key={a.id} className="rounded-xl border border-neutral-200/90 bg-white p-5 shadow-2xs flex flex-col justify-between">
              <div>
                <div className="flex items-center justify-between gap-3 mb-3">
                  <span className="text-[11px] font-bold uppercase tracking-wider px-2 py-0.5 rounded bg-neutral-100 text-neutral-700 border border-neutral-200/60">
                    {a.brand_name}
                  </span>
                  <span
                    className={`rounded-full px-2.5 py-0.5 text-[11px] font-bold ${
                      a.is_active
                        ? 'bg-emerald-50 text-emerald-700 border border-emerald-200/60'
                        : 'bg-neutral-100 text-neutral-500 border border-neutral-200/60'
                    }`}
                  >
                    {a.is_active ? 'Aktif' : 'Nonaktif'}
                  </span>
                </div>

                <div className="flex items-start gap-3.5 pt-1">
                  {logoSrc ? (
                    <div className="w-12 h-12 rounded-lg border border-neutral-200 bg-white p-1 flex items-center justify-center shrink-0 shadow-2xs">
                      <img
                        src={logoSrc}
                        alt={a.bank_name}
                        className="max-h-full max-w-full object-contain"
                      />
                    </div>
                  ) : (
                    <div className="w-12 h-12 rounded-lg border border-neutral-200 bg-neutral-100 flex items-center justify-center text-neutral-400 font-black text-[10px] tracking-wider shrink-0">
                      BANK
                    </div>
                  )}

                  <div className="min-w-0 flex-1">
                    <h2 className="text-base font-bold text-neutral-900 truncate">{a.bank_name}</h2>
                    <p className="font-mono text-sm font-bold text-neutral-800 tracking-wider mt-0.5">{a.account_number}</p>
                    <p className="text-xs text-neutral-500 mt-0.5 truncate">a.n. {a.account_holder}</p>
                  </div>
                </div>

                {a.instructions && (
                  <p className="mt-3 text-xs text-neutral-500 bg-neutral-50 p-2.5 rounded-lg border border-neutral-100 line-clamp-2">
                    {a.instructions}
                  </p>
                )}
              </div>

              <div className="mt-5 flex items-center justify-end gap-2 border-t border-neutral-100 pt-3.5">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => handleOpenEdit(a)}
                >
                  Edit
                </Button>
                <Button
                  type="button"
                  variant="danger"
                  size="sm"
                  onClick={async () => {
                    if (confirm(`Hapus rekening ${a.bank_name} - ${a.account_number}?`)) {
                      await deleteBankAccount(a.id);
                      load();
                    }
                  }}
                >
                  Hapus
                </Button>
              </div>
            </article>
          );
        })}
      </div>

      <Modal
        isOpen={open}
        onClose={() => setOpen(false)}
        title={editing ? 'Edit Rekening Bank' : 'Tambah Rekening Bank'}
      >
        <form onSubmit={save} className="space-y-4">
          <FormField label="Brand Travel" required>
            <select
              required
              value={form.brand_id}
              onChange={(e) => setForm({ ...form, brand_id: e.target.value })}
              className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm focus:border-neutral-900 focus:outline-none"
            >
              <option value="">Pilih brand</option>
              {brands.map((b) => (
                <option key={b.id} value={b.id}>
                  {b.name}
                </option>
              ))}
            </select>
          </FormField>

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
            onChange={handleLogoFileChange}
            onRemove={() => {
              setLogoPreview(null);
              setForm((prev) => ({ ...prev, logo_url: '' }));
            }}
            isUploading={isUploadingLogo}
            uploadingText="Mengunggah logo bank..."
            placeholder="Pilih file logo bank (PNG, JPG, SVG)..."
            helperText="Format: PNG, JPG, WEBP, SVG. Disarankan gambar transparan."
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
            value={form.instructions || ''}
            onChange={(e) => setForm({ ...form, instructions: e.target.value })}
          />

          <label className="flex items-center gap-2 text-sm text-neutral-700 cursor-pointer pt-1">
            <input
              type="checkbox"
              checked={form.is_active}
              onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
              className="rounded border-neutral-300 text-neutral-900 focus:ring-0 cursor-pointer"
            />
            <span>Aktif dan tampil di formulir booking / portal jamaah</span>
          </label>

          <div className="flex justify-end gap-2 pt-3 border-t border-neutral-100">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
              Batal
            </Button>
            <Button type="submit" disabled={isUploadingLogo}>
              Simpan
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
