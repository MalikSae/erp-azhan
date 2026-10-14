import React, { useContext, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Home, ChevronRight, ChevronDown, Plus, X, Power, Trash2, KeyRound } from 'lucide-react';
import Button from 'shared/ui/Button';
import Alert from 'shared/ui/Alert';
import Input from 'shared/ui/Input';
import Modal from 'shared/ui/Modal';
import CustomDropdown from 'shared/ui/CustomDropdown';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import { AuthContext } from '../context/AuthContext';
import {
  getAdminUser, createAdminUser, updateAdminUser,
  setAdminUserStatus, resetAdminUserPassword, deleteAdminUser,
} from '../api/adminUsers';
import { listRoles } from '../api/roles';
import { listBrands } from '../api/brands';
import { listOrgUnits } from '../api/orgUnits';
import { errMsg, Panel, StatusChip } from '../components/penggunaUi';

const emptyForm = {
  display_name: '', email: '', no_hp: '', jabatan: '',
  access: 'holding', brand_id: '', password: '',
};

const roleFitsAccess = (role, access) => role.scope === 'semua' || role.scope === access;
const SUPER_ADMIN = 'super_admin_grup';

// R10: divisi yang boleh dipilih di Penugasan mengikuti divisi asal role.
// 'all' = Super Admin (unit mana pun); akses Brand tidak masuk struktur Holding.
const allowedDivisi = (roles, slugs, access) => {
  if (access !== 'holding') return new Set();
  if (slugs.includes(SUPER_ADMIN)) return 'all';
  return new Set(roles.filter(r => slugs.includes(r.slug) && r.divisi_id).map(r => r.divisi_id));
};

// Form Pengguna halaman penuh (SPEK-PENGGUNA-ORGANISASI §6): Profil, Akses,
// Role, Penugasan, Keamanan — disimpan atomik dalam satu permintaan.
const PenggunaFormPage = () => {
  const { id } = useParams();
  const isEdit = Boolean(id);
  const navigate = useNavigate();
  const { currentUserId } = useContext(AuthContext);
  const isSelf = isEdit && currentUserId && Number(id) === Number(currentUserId);

  const [roles, setRoles] = useState([]);
  const [brands, setBrands] = useState([]);
  const [units, setUnits] = useState([]); // pohon divisi -> tim
  const [user, setUser] = useState(null); // data asli (mode edit)
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState(null);

  const [form, setForm] = useState(emptyForm);
  const [selectedRoles, setSelectedRoles] = useState([]);
  const [keanggotaan, setKeanggotaan] = useState([]); // [{org_unit_id, posisi}]
  const [expandedRole, setExpandedRole] = useState(null);
  const [roleNotice, setRoleNotice] = useState(null);
  const [penugasanNotice, setPenugasanNotice] = useState(null);
  const [submitError, setSubmitError] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Tindakan akun (mode edit)
  const [actionModal, setActionModal] = useState(null); // 'status' | 'delete' | 'password'
  const [actionError, setActionError] = useState(null);
  const [isActing, setIsActing] = useState(false);
  const [newPassword, setNewPassword] = useState('');

  useEffect(() => {
    let active = true;
    const load = async () => {
      try {
        const [rolesData, brandsData, unitsData, userData] = await Promise.all([
          listRoles(), listBrands(), listOrgUnits(), isEdit ? getAdminUser(id) : Promise.resolve(null),
        ]);
        if (!active) return;
        setRoles(rolesData || []);
        setBrands(brandsData || []);
        setUnits(unitsData || []);
        if (userData) {
          setUser(userData);
          const holding = userData.brand_id === null || userData.brand_id === undefined;
          setForm({
            display_name: userData.display_name || '',
            email: userData.email || '',
            no_hp: userData.no_hp || '',
            jabatan: userData.jabatan || '',
            access: holding ? 'holding' : 'brand',
            brand_id: holding ? '' : String(userData.brand_id),
            password: '',
          });
          setSelectedRoles(userData.roles || []);
          setKeanggotaan((userData.keanggotaan || []).map(k => ({ org_unit_id: k.org_unit_id, posisi: k.posisi })));
        }
      } catch (error) {
        if (active) setLoadError(errMsg(error, 'Gagal memuat data.'));
      } finally {
        if (active) setIsLoading(false);
      }
    };
    load();
    return () => { active = false; };
  }, [id, isEdit]);

  // ─── Role ────────────────────────────────────────────────────────────────
  const visibleRoles = useMemo(() => roles.filter(r => roleFitsAccess(r, form.access)), [roles, form.access]);

  // Penugasan yang tak lagi sesuai role/akses dilepas — beri tahu, jangan diam-diam.
  const pruneKeanggotaan = (nextSlugs, access) => {
    const allowed = allowedDivisi(roles, nextSlugs, access);
    if (allowed === 'all') return;
    const divisiOf = (unitId) => {
      for (const d of units) {
        if (d.id === unitId) return d.id;
        if ((d.tim || []).some(t => t.id === unitId)) return d.id;
      }
      return null;
    };
    const dropped = keanggotaan.filter(k => !allowed.has(divisiOf(k.org_unit_id)));
    if (!dropped.length) return;
    setKeanggotaan(prev => prev.filter(k => allowed.has(divisiOf(k.org_unit_id))));
    const names = dropped.map(k => unitLabel(k.org_unit_id)).join(', ');
    setPenugasanNotice(access !== 'holding'
      ? `Penugasan ${names} dilepas — pengguna akses Brand tidak masuk struktur organisasi Holding.`
      : `Penugasan ${names} dilepas karena bukan divisi dari role yang dipilih.`);
  };

  const changeAccess = (access) => {
    if (access === form.access) return;
    setForm(p => ({ ...p, access, brand_id: '' }));
    // Role yang tidak sesuai akses baru dilepas — beri tahu, jangan diam-diam.
    const dropped = selectedRoles.filter(slug => {
      const r = roles.find(x => x.slug === slug);
      return r && !roleFitsAccess(r, access);
    });
    const nextSlugs = selectedRoles.filter(s => !dropped.includes(s));
    if (dropped.length) {
      setSelectedRoles(nextSlugs);
      const names = dropped.map(s => roles.find(r => r.slug === s)?.name || s).join(', ');
      setRoleNotice(`Role ${names} dilepas karena tidak sesuai dengan akses ${access === 'holding' ? 'Holding' : 'Brand'}.`);
    } else {
      setRoleNotice(null);
    }
    pruneKeanggotaan(nextSlugs, access);
  };

  const toggleRole = (slug) => {
    const next = selectedRoles.includes(slug) ? selectedRoles.filter(s => s !== slug) : [...selectedRoles, slug];
    setSelectedRoles(next);
    pruneKeanggotaan(next, form.access);
  };

  const totalPermissions = useMemo(() => {
    const set = new Set();
    roles.filter(r => selectedRoles.includes(r.slug)).forEach(r => (r.permissions || []).forEach(p => set.add(p)));
    return set.size;
  }, [roles, selectedRoles]);

  // ─── Penugasan ───────────────────────────────────────────────────────────
  // Opsi unit: unit aktif + unit lama milik pengguna walau kini nonaktif
  // (boleh dipertahankan, R9).
  const originalUnitIds = useMemo(() => new Set((user?.keanggotaan || []).map(k => k.org_unit_id)), [user]);
  const allowed = useMemo(() => allowedDivisi(roles, selectedRoles, form.access), [roles, selectedRoles, form.access]);
  const unitLabel = (unitId) => {
    for (const d of units) {
      if (d.id === unitId) return d.nama;
      const t = (d.tim || []).find(x => x.id === unitId);
      if (t) return `${d.nama} › ${t.nama}`;
    }
    return `#${unitId}`;
  };
  const unitOptions = useMemo(() => {
    const opts = [];
    units.forEach(d => {
      if (allowed !== 'all' && !allowed.has(d.id)) return;
      if (d.is_active || originalUnitIds.has(d.id)) {
        opts.push({ id: d.id, label: `${d.nama}${d.is_active ? '' : ' (nonaktif)'}`, jenis: 'divisi' });
      }
      (d.tim || []).forEach(t => {
        if (t.is_active || originalUnitIds.has(t.id)) {
          opts.push({ id: t.id, label: `${d.nama} › ${t.nama}${t.is_active ? '' : ' (nonaktif)'}`, jenis: 'tim' });
        }
      });
    });
    return opts;
  }, [units, originalUnitIds, allowed]);

  const allowedNames = allowed === 'all'
    ? null
    : units.filter(d => allowed.has(d.id)).map(d => d.nama);

  const optionsForRow = (rowIndex) => {
    const takenElsewhere = new Set(keanggotaan.filter((_, i) => i !== rowIndex).map(k => k.org_unit_id));
    return unitOptions
      .filter(o => !takenElsewhere.has(o.id))
      .map(o => ({ value: String(o.id), label: o.label }));
  };

  const addRow = () => {
    const taken = new Set(keanggotaan.map(k => k.org_unit_id));
    const first = unitOptions.find(o => !taken.has(o.id));
    if (!first) return;
    setKeanggotaan(prev => [...prev, { org_unit_id: first.id, posisi: 'anggota' }]);
  };
  const updateRow = (i, patch) => setKeanggotaan(prev => prev.map((k, idx) => (idx === i ? { ...k, ...patch } : k)));
  const removeRow = (i) => setKeanggotaan(prev => prev.filter((_, idx) => idx !== i));

  // ─── Simpan ──────────────────────────────────────────────────────────────
  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitError(null);
    if (form.access === 'brand' && !form.brand_id) { setSubmitError('Pilih brand untuk akses Brand.'); return; }
    if (selectedRoles.length === 0) { setSubmitError('Pilih minimal satu role.'); return; }
    if (!isEdit && form.password.trim().length < 8) { setSubmitError('Password minimal 8 karakter.'); return; }

    const payload = {
      display_name: form.display_name.trim(),
      email: form.email.trim(),
      no_hp: form.no_hp.trim() || null,
      jabatan: form.jabatan.trim() || null,
      brand_id: form.access === 'holding' ? null : Number(form.brand_id),
      roles: selectedRoles,
      keanggotaan,
    };

    setIsSubmitting(true);
    try {
      if (isEdit) {
        await updateAdminUser(id, payload);
        navigate('/users', { state: { successMessage: `Pengguna "${payload.display_name}" berhasil diperbarui.` } });
      } else {
        await createAdminUser({ ...payload, password: form.password.trim() });
        navigate('/users', { state: { successMessage: `Pengguna "${payload.display_name}" berhasil ditambahkan.` } });
      }
    } catch (error) {
      setSubmitError(errMsg(error, 'Gagal menyimpan pengguna.'));
      window.scrollTo?.({ top: 0, behavior: 'smooth' });
      document.querySelector('main')?.scrollTo?.({ top: 0, behavior: 'smooth' });
    } finally {
      setIsSubmitting(false);
    }
  };

  // ─── Tindakan akun (edit) ────────────────────────────────────────────────
  const runAction = async () => {
    setIsActing(true);
    setActionError(null);
    try {
      if (actionModal === 'status') {
        const next = !user.is_active;
        const updated = await setAdminUserStatus(id, next);
        setUser(updated);
        setActionModal(null);
      } else if (actionModal === 'delete') {
        await deleteAdminUser(id);
        navigate('/users', { state: { successMessage: `Pengguna "${user.display_name}" dihapus permanen.` } });
      } else if (actionModal === 'password') {
        if (newPassword.trim().length < 8) { setActionError('Password minimal 8 karakter.'); setIsActing(false); return; }
        await resetAdminUserPassword(id, newPassword.trim());
        setActionModal(null);
        setNewPassword('');
      }
    } catch (error) {
      setActionError(errMsg(error, 'Aksi gagal.'));
    } finally {
      setIsActing(false);
    }
  };

  const title = isEdit ? 'Edit Pengguna' : 'Tambah Pengguna';

  if (isLoading) {
    return <div className="flex justify-center p-10 bg-white rounded-2xl border border-neutral-200 shadow-card"><LoadingSpinner /></div>;
  }

  return (
    <>
    <form onSubmit={handleSubmit} className="space-y-5">
      {/* Judul + breadcrumb */}
      <div>
        <h1 className="text-lg md:text-xl font-heading font-bold text-neutral-900 tracking-tight">{title}</h1>
        <div className="mt-1 flex items-center gap-1.5 text-xs font-body text-neutral-500">
          <Home className="w-3.5 h-3.5" />
          <span>Dashboard</span>
          <ChevronRight className="w-3 h-3 text-neutral-300" />
          <button type="button" className="hover:text-neutral-700" onClick={() => navigate('/users')}>Pengguna</button>
          <ChevronRight className="w-3 h-3 text-neutral-300" />
          <span className="text-neutral-700 font-medium">{isEdit ? (user?.display_name || title) : title}</span>
        </div>
      </div>

      {loadError && <Alert variant="error">{loadError}</Alert>}
      {submitError && <Alert variant="error" onClose={() => setSubmitError(null)}>{submitError}</Alert>}

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-4 items-start">
        {/* ── Kolom kiri ── */}
        <div className="lg:col-span-5 space-y-4">
          <Panel title="Profil">
            <Input label="Nama" required value={form.display_name} placeholder="Nama lengkap"
              onChange={(e) => setForm(p => ({ ...p, display_name: e.target.value }))} autoFocus={!isEdit} />
            <Input label="Email" type="email" required value={form.email} placeholder="nama@contoh.com"
 onChange={(e) => setForm(p => ({ ...p, email: e.target.value }))} />
            <div className="grid grid-cols-2 gap-3">
              <Input label="No. HP" value={form.no_hp} placeholder="08xxxxxxxxxx"
                onChange={(e) => setForm(p => ({ ...p, no_hp: e.target.value }))} />
              <Input label="Jabatan" value={form.jabatan} placeholder="mis. Direktur"
                onChange={(e) => setForm(p => ({ ...p, jabatan: e.target.value }))} />
            </div>
            <p className="text-[11px] font-body text-neutral-400 -mt-1">Jabatan hanya label tampilan — tidak memengaruhi akses.</p>
          </Panel>

          <Panel title="Akses Brand">
            <div className={form.access === 'brand' ? 'grid grid-cols-2 gap-3' : ''}>
              <CustomDropdown
                label="Cakupan data"
                required
                value={form.access}
                onChange={changeAccess}
                options={[
                  { value: 'holding', label: 'Holding (semua brand)' },
                  { value: 'brand', label: 'Brand tertentu' },
                ]}
              />
              {form.access === 'brand' ? (
                <CustomDropdown
                  label="Brand"
                  required
                  value={form.brand_id}
                  onChange={(val) => setForm(p => ({ ...p, brand_id: val }))}
                  placeholder="Pilih brand..."
                  options={brands.map(b => ({ value: String(b.id), label: b.name }))}
                />
              ) : null}
            </div>
            {!isEdit && (
              <Input label="Password" type="password" required value={form.password}
                onChange={(e) => setForm(p => ({ ...p, password: e.target.value }))} />
            )}
          </Panel>

        </div>

        {/* ── Kolom kanan ── */}
        <div className="lg:col-span-7 space-y-4">
          <Panel
            title="Role"
            bodyClass="p-0"
            right={<span className="text-xs font-body text-neutral-500">{selectedRoles.length} dipilih · {totalPermissions} permission</span>}
          >
            {roleNotice && <div className="px-5 pt-4"><Alert variant="warning" onClose={() => setRoleNotice(null)}>{roleNotice}</Alert></div>}
            <p className="px-5 pt-4 text-[11px] font-body text-neutral-400">
              Hanya role yang sesuai akses {form.access === 'holding' ? 'Holding' : 'Brand'} yang ditampilkan. Klik baris untuk melihat permission.
            </p>
            <ul className="divide-y divide-neutral-100 mt-2 rounded-b-2xl overflow-hidden">
              {visibleRoles.map(role => {
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
                        className="h-4 w-4 rounded border-neutral-300 accent-primary-500 shrink-0"
                        checked={checked}
                        onChange={() => toggleRole(role.slug)}
                        onClick={(e) => e.stopPropagation()}
                      />
                      <div className="flex-1 min-w-0">
                        <p className={`text-[13px] font-heading font-semibold truncate ${checked ? 'text-primary-600' : 'text-neutral-900'}`}>{role.name}</p>
                        <p className="text-[11px] font-body text-neutral-400 truncate">
                          {role.slug}
                          {role.slug === SUPER_ADMIN ? ' · semua divisi' : role.divisi_nama ? ` · divisi ${role.divisi_nama}` : ''}
                        </p>
                      </div>
                      <span className="text-[11px] font-body text-neutral-400 shrink-0">
                        {(role.permissions || []).length} permission
                      </span>
                      <ChevronDown className={`w-4 h-4 text-neutral-400 transition-transform shrink-0 ${expanded ? 'rotate-180' : ''}`} />
                    </div>
                    {expanded && (
                      <div className="px-5 pb-4 pt-1 bg-neutral-50/50 flex flex-wrap gap-1.5">
                        {(role.permissions || []).map(perm => (
                          <span key={perm} className="text-[11px] font-mono text-neutral-600 bg-white border border-neutral-200 rounded-md px-2 py-0.5">{perm}</span>
                        ))}
                      </div>
                    )}
                  </li>
                );
              })}
            </ul>
          </Panel>

          <Panel
            title="Penugasan"
            right={(
              form.access === 'holding' && (
                <Button type="button" variant="secondary" size="sm" onClick={addRow} disabled={keanggotaan.length >= unitOptions.length}>
                  <Plus className="w-3.5 h-3.5" /> Tambah
                </Button>
              )
            )}
          >
            {penugasanNotice && <div className="mb-3"><Alert variant="warning" onClose={() => setPenugasanNotice(null)}>{penugasanNotice}</Alert></div>}
            {form.access !== 'holding' ? (
              <p className="text-xs font-body text-neutral-500">
                Pengguna akses Brand tidak masuk struktur organisasi Holding, jadi tidak punya penugasan.
              </p>
            ) : unitOptions.length === 0 ? (
              <p className="text-xs font-body text-neutral-500">
                Pilih role dulu — unit yang bisa dipilih mengikuti divisi role.
              </p>
            ) : keanggotaan.length === 0 ? (
              <p className="text-xs font-body text-neutral-500">
                Belum ditugaskan ke unit mana pun. Unit yang bisa dipilih: {allowedNames ? allowedNames.join(', ') : 'semua divisi'} beserta timnya.
              </p>
            ) : (
              <div className="space-y-2">
                {keanggotaan.map((k, i) => (
                  <div key={i} className="flex items-end gap-2">
                    <div className="flex-1 min-w-0">
                      <CustomDropdown
                        label={i === 0 ? 'Unit' : undefined}
                        value={String(k.org_unit_id)}
                        onChange={(val) => updateRow(i, { org_unit_id: Number(val) })}
                        options={optionsForRow(i)}
                      />
                    </div>
                    <div className="w-36">
                      <CustomDropdown
                        label={i === 0 ? 'Posisi' : undefined}
                        value={k.posisi}
                        onChange={(val) => updateRow(i, { posisi: val })}
                        options={[{ value: 'anggota', label: 'Anggota' }, { value: 'pemimpin', label: 'Pemimpin' }]}
                      />
                    </div>
                    <button type="button" title="Hapus baris" onClick={() => removeRow(i)}
                      className="mb-4 w-10 h-10 shrink-0 flex items-center justify-center rounded-lg border border-neutral-200 text-neutral-500 hover:text-danger-600 hover:bg-danger-50 hover:border-danger-200">
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </Panel>

          {isEdit && user && (
            <Panel title="Tindakan Akun" right={<StatusChip active={user.is_active} />}>
              <div className="space-y-3">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <p className="text-[13px] font-heading font-semibold text-neutral-900">Ganti password</p>
                    <p className="text-xs font-body text-neutral-500">Sesi lama pengguna ini dicabut.</p>
                  </div>
                  <Button type="button" variant="secondary" size="sm" disabled={!user.is_active}
                    onClick={() => { setActionModal('password'); setActionError(null); setNewPassword(''); }}>
                    <KeyRound className="w-3.5 h-3.5" /> Ganti
                  </Button>
                </div>
                <div className="flex items-start justify-between gap-4 pt-3 border-t border-neutral-100">
                  <div>
                    <p className="text-[13px] font-heading font-semibold text-neutral-900">{user.is_active ? 'Nonaktifkan pengguna' : 'Aktifkan kembali'}</p>
                    <p className="text-xs font-body text-neutral-500">
                      {user.is_active ? 'Cara offboarding: login ditolak & semua sesi dicabut. Riwayat tetap utuh.' : 'Pengguna bisa login lagi dengan role & akses tersimpan.'}
                    </p>
                  </div>
                  <Button type="button" variant={user.is_active ? 'danger-light' : 'secondary'} size="sm" disabled={isSelf}
                    title={isSelf ? 'Tidak bisa menonaktifkan akun sendiri' : undefined}
                    onClick={() => { setActionModal('status'); setActionError(null); }}>
                    <Power className="w-3.5 h-3.5" /> {user.is_active ? 'Nonaktifkan' : 'Aktifkan'}
                  </Button>
                </div>
                <div className="flex items-start justify-between gap-4 pt-3 border-t border-neutral-100">
                  <div>
                    <p className="text-[13px] font-heading font-semibold text-neutral-900">Hapus permanen</p>
                    <p className="text-xs font-body text-neutral-500">Hanya untuk akun salah buat yang belum punya riwayat. Selebihnya, nonaktifkan.</p>
                  </div>
                  <Button type="button" variant="danger-light" size="sm" disabled={isSelf}
                    onClick={() => { setActionModal('delete'); setActionError(null); }}>
                    <Trash2 className="w-3.5 h-3.5" /> Hapus
                  </Button>
                </div>
              </div>
            </Panel>
          )}
        </div>
      </div>

      {/* Bar aksi menempel di bawah */}
      <div className="sticky bottom-0 -mx-4 md:-mx-8 !-mb-4 md:!-mb-8 px-4 md:px-8 py-3 bg-white/95 backdrop-blur border-t border-neutral-200 flex justify-end gap-3 z-10">
        <Button type="button" variant="secondary" onClick={() => navigate('/users')} disabled={isSubmitting}>Batal</Button>
        <Button type="submit" variant="primary" disabled={isSubmitting || Boolean(loadError)}>
          {isSubmitting ? 'Menyimpan...' : (isEdit ? 'Simpan Perubahan' : 'Simpan Pengguna')}
        </Button>
      </div>

    </form>

      {/* Modal tindakan akun — di LUAR form utama supaya Enter tidak men-submit form pengguna */}
      <Modal
        isOpen={!!actionModal}
        onClose={() => { if (!isActing) setActionModal(null); }}
        title={actionModal === 'delete' ? 'Hapus Pengguna Permanen' : actionModal === 'password' ? 'Ganti Password' : (user?.is_active ? 'Nonaktifkan Pengguna' : 'Aktifkan Kembali')}
      >
        <div className="space-y-4">
          {actionError && <Alert variant="error">{actionError}</Alert>}
          {actionModal === 'password' && (
            <Input label="Password Baru" type="password" required value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)} autoFocus />
          )}
          {actionModal === 'status' && (
            <p className="text-sm font-body text-neutral-700">
              {user?.is_active
                ? <>Nonaktifkan <strong className="font-heading text-neutral-900">{user?.display_name}</strong>? Login ditolak dan semua sesinya langsung dicabut.</>
                : <>Aktifkan kembali <strong className="font-heading text-neutral-900">{user?.display_name}</strong>?</>}
            </p>
          )}
          {actionModal === 'delete' && (
            <p className="text-sm font-body text-neutral-700">
              Hapus <strong className="font-heading text-neutral-900">{user?.display_name}</strong> secara permanen? Ditolak otomatis bila pengguna ini punya riwayat transaksi — gunakan Nonaktifkan.
            </p>
          )}
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button type="button" variant="secondary" onClick={() => setActionModal(null)} disabled={isActing}>Batal</Button>
            <Button type="button" variant={actionModal === 'password' || (actionModal === 'status' && !user?.is_active) ? 'primary' : 'danger'} onClick={runAction} disabled={isActing}>
              {isActing ? 'Memproses...' : actionModal === 'delete' ? 'Hapus Permanen' : actionModal === 'password' ? 'Simpan Password' : (user?.is_active ? 'Nonaktifkan' : 'Aktifkan')}
            </Button>
          </div>
        </div>
      </Modal>
    </>
  );
};

export default PenggunaFormPage;
