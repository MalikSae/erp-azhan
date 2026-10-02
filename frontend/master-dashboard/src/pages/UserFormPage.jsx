import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Home, ChevronRight, ChevronDown } from 'lucide-react';
import Button from '../components/ui/Button';
import Alert from '../components/ui/Alert';
import FormField from '../components/ui/FormField';
import Input from '../components/ui/Input';
import CustomDropdown from '../components/ui/CustomDropdown';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import { createAdminUser } from '../api/adminUsers';
import { listBrands } from '../api/brands';
import { listRoles, setUserRoles } from '../api/roles';

// Akses brand menentukan scope data (brand_id), terpisah dari role RBAC.
const accessOptions = [
  { value: 'holding', label: 'Holding (semua brand)' },
  { value: 'brand', label: 'Per Brand' },
];

const defaultRolesForAccess = (access) =>
  access === 'holding' ? ['super_admin_grup'] : ['admin_travel'];

// Halaman penuh Tambah User: informasi akun di kiri, pemilih role RBAC di
// kanan — tiap role bisa dibuka untuk melihat seluruh permission-nya.
const UserFormPage = () => {
  const navigate = useNavigate();

  const [brands, setBrands] = useState([]);
  const [roles, setRoles] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState(null);

  const [form, setForm] = useState({ email: '', password: '', access: 'holding', brand_id: '' });
  const [selectedRoles, setSelectedRoles] = useState(defaultRolesForAccess('holding'));
  const [expandedRole, setExpandedRole] = useState(null);
  const [submitError, setSubmitError] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    let active = true;
    Promise.all([listBrands(), listRoles()])
      .then(([brandsData, rolesData]) => {
        if (active) { setBrands(brandsData || []); setRoles(rolesData || []); }
      })
      .catch((err) => { if (active) setLoadError(err.response?.data?.error || 'Gagal memuat data role/brand.'); })
      .finally(() => { if (active) setIsLoading(false); });
    return () => { active = false; };
  }, []);

  const toggleRole = (slug) => {
    setSelectedRoles((prev) => prev.includes(slug) ? prev.filter((s) => s !== slug) : [...prev, slug]);
  };

  const totalPermissions = useMemo(() => {
    const set = new Set();
    roles.filter((r) => selectedRoles.includes(r.slug)).forEach((r) => (r.permissions || []).forEach((p) => set.add(p)));
    return set.size;
  }, [roles, selectedRoles]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitError(null);

    const email = form.email.trim();
    if (!email) { setSubmitError('Email wajib diisi.'); return; }
    if (form.password.trim().length < 8) { setSubmitError('Password minimal 8 karakter.'); return; }
    if (form.access === 'brand' && !form.brand_id) { setSubmitError('Brand wajib dipilih untuk akses per brand.'); return; }
    if (selectedRoles.length === 0) { setSubmitError('Minimal satu role wajib dipilih.'); return; }

    setIsSubmitting(true);
    try {
      const created = await createAdminUser({
        email,
        password: form.password.trim(),
        brand_id: form.access === 'holding' ? null : Number(form.brand_id),
      });
      if (created?.id) {
        try {
          await setUserRoles(created.id, selectedRoles);
        } catch (roleErr) {
          navigate('/users', { state: { errorMessage: `User dibuat, tetapi gagal mengatur role: ${roleErr.response?.data?.error || 'atur lewat Kelola Role.'}` } });
          return;
        }
      }
      navigate('/users', { state: { successMessage: `User "${email}" berhasil ditambahkan.` } });
    } catch (error) {
      setSubmitError(error.response?.data?.error || 'Gagal menambahkan user baru.');
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-5 pb-10">
      {/* Judul + breadcrumb */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3">
        <div>
          <h1 className="text-lg md:text-xl font-heading font-bold text-neutral-900 tracking-tight">Tambah User</h1>
          <div className="mt-1 flex items-center gap-1.5 text-xs font-body text-neutral-500">
            <Home className="w-3.5 h-3.5" />
            <span>Dashboard</span>
            <ChevronRight className="w-3 h-3 text-neutral-300" />
            <span className="cursor-pointer hover:text-neutral-700" onClick={() => navigate('/users')}>User Management</span>
            <ChevronRight className="w-3 h-3 text-neutral-300" />
            <span className="text-neutral-700 font-medium">Tambah User</span>
          </div>
        </div>
      </div>

      {loadError && <Alert variant="error">{loadError}</Alert>}
      {submitError && <Alert variant="error">{submitError}</Alert>}

      {isLoading ? (
        <div className="flex justify-center p-10 bg-white rounded-2xl border border-neutral-200 shadow-card"><LoadingSpinner /></div>
      ) : (
        <form onSubmit={handleSubmit} className="grid grid-cols-1 lg:grid-cols-12 gap-4 items-start">
          {/* Kiri: informasi akun */}
          <section className="lg:col-span-5 bg-white rounded-2xl border border-neutral-200 shadow-card overflow-hidden">
            <div className="px-5 py-4 border-b border-neutral-100">
              <h3 className="text-[15px] font-heading font-bold text-neutral-900">Informasi Akun</h3>
            </div>
            <div className="p-5 space-y-4">
              <FormField label="Email Akun" required>
                <Input type="email" value={form.email} onChange={(e) => setForm((p) => ({ ...p, email: e.target.value }))} placeholder="nama@azhan.id" autoFocus required />
              </FormField>
              <FormField label="Password" hint="Minimal 8 karakter." required>
                <Input type="password" value={form.password} onChange={(e) => setForm((p) => ({ ...p, password: e.target.value }))} placeholder="••••••••" required />
              </FormField>
              <CustomDropdown
                label="Akses Brand"
                required
                value={form.access}
                onChange={(val) => { setForm((p) => ({ ...p, access: val, brand_id: '' })); setSelectedRoles(defaultRolesForAccess(val)); }}
                options={accessOptions}
              />
              {form.access === 'brand' && (
                <CustomDropdown
                  label="Pilih Brand"
                  required
                  value={form.brand_id}
                  onChange={(val) => setForm((p) => ({ ...p, brand_id: val }))}
                  placeholder="Pilih Brand..."
                  options={[{ value: '', label: 'Pilih Brand...' }, ...brands.map((b) => ({ value: String(b.id), label: b.name }))]}
                />
              )}

              <div className="rounded-xl bg-neutral-50 border border-neutral-100 px-4 py-3 text-xs font-body text-neutral-600">
                {selectedRoles.length} role dipilih · gabungan <span className="font-semibold text-neutral-900">{totalPermissions} permission</span>
              </div>
            </div>
            <div className="px-5 py-4 border-t border-neutral-100 flex justify-end gap-2.5">
              <Button type="button" variant="secondary" onClick={() => navigate('/users')} disabled={isSubmitting}>Batal</Button>
              <Button type="submit" variant="primary" disabled={isSubmitting}>{isSubmitting ? 'Menyimpan...' : 'Simpan User'}</Button>
            </div>
          </section>

          {/* Kanan: pemilih role + permission viewer */}
          <section className="lg:col-span-7 bg-white rounded-2xl border border-neutral-200 shadow-card overflow-hidden">
            <div className="px-5 py-4 border-b border-neutral-100 flex items-center justify-between">
              <h3 className="text-[15px] font-heading font-bold text-neutral-900">Role (RBAC)</h3>
              <span className="text-xs font-body text-neutral-400">klik baris untuk melihat permission</span>
            </div>
            <ul className="divide-y divide-neutral-100">
              {roles.map((role) => {
                const checked = selectedRoles.includes(role.slug);
                const expanded = expandedRole === role.slug;
                return (
                  <li key={role.slug}>
                    <div
                      className={`flex items-center gap-3 px-5 py-3 cursor-pointer transition-colors ${checked ? 'bg-primary-50/60' : 'hover:bg-neutral-50'}`}
                      onClick={() => setExpandedRole(expanded ? null : role.slug)}
                    >
                      <input
                        type="checkbox"
                        className="h-4 w-4 rounded border-neutral-300 accent-[#F26522] shrink-0"
                        checked={checked}
                        onChange={() => toggleRole(role.slug)}
                        onClick={(e) => e.stopPropagation()}
                      />
                      <div className="flex-1 min-w-0">
                        <p className={`text-[13px] font-heading font-semibold truncate ${checked ? 'text-primary-600' : 'text-neutral-900'}`}>{role.name}</p>
                        <p className="text-[11px] font-body text-neutral-400 truncate">{role.slug}</p>
                      </div>
                      <span className="text-[11px] font-medium font-body text-neutral-500 bg-neutral-100 rounded-md px-2 py-0.5 shrink-0">
                        {(role.permissions || []).length} permission
                      </span>
                      <ChevronDown className={`w-4 h-4 text-neutral-400 transition-transform shrink-0 ${expanded ? 'rotate-180' : ''}`} />
                    </div>
                    {expanded && (
                      <div className="px-5 pb-4 pt-1 bg-neutral-50/50">
                        {(role.permissions || []).length === 0 ? (
                          <p className="text-xs font-body text-neutral-400 italic">Role ini belum memiliki permission.</p>
                        ) : (
                          <div className="flex flex-wrap gap-1.5">
                            {role.permissions.map((perm) => (
                              <span key={perm} className="text-[11px] font-mono text-neutral-600 bg-white border border-neutral-200 rounded-md px-2 py-0.5">
                                {perm}
                              </span>
                            ))}
                          </div>
                        )}
                      </div>
                    )}
                  </li>
                );
              })}
            </ul>
          </section>
        </form>
      )}
    </div>
  );
};

export default UserFormPage;
