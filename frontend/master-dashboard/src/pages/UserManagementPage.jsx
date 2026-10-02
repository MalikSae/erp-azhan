import React, { useState, useEffect, useContext } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { Users, ShieldCheck, Building2, KeyRound } from 'lucide-react';
import PageHeader from '../components/ui/PageHeader';
import DataTable from '../components/ui/DataTable';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import FormField from '../components/ui/FormField';
import Input from '../components/ui/Input';
import CustomDropdown from '../components/ui/CustomDropdown';
import Badge from '../components/ui/Badge';
import Alert from '../components/ui/Alert';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import StatTile from '../components/ui/StatTile';
import { AuthContext } from '../context/AuthContext';
import {
  listAdminUsers,
  updateAdminUser,
  resetAdminUserPassword,
  deleteAdminUser
} from '../api/adminUsers';
import { listBrands } from '../api/brands';
import { listRoles, setUserRoles } from '../api/roles';

const initialEditForm = {
  email: '',
  role: 'super_admin',
  brand_id: ''
};

// Akses brand menentukan scope data (brand_id), terpisah dari role RBAC.
const roleOptions = [
  { value: 'super_admin', label: 'Holding (semua brand)' },
  { value: 'travel_admin', label: 'Per Brand' }
];

// Kolom "Akses Brand" cukup menampilkan nama brand (tanpa awalan "Admin").
const getBrandAccessLabel = (user) => {
  if (user.brand_id === null || user.brand_id === undefined) {
    return 'Semua Brand';
  }
  return user.brand_name || `Brand #${user.brand_id}`;
};

const UserManagementPage = () => {
  const { currentUserId } = useContext(AuthContext);
  const navigate = useNavigate();
  const location = useLocation();

  const [users, setUsers] = useState([]);
  const [brands, setBrands] = useState([]);
  const [roles, setRoles] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  const [successMessage, setSuccessMessage] = useState(null);

  // Modal Kelola Role (RBAC)
  const [roleTargetUser, setRoleTargetUser] = useState(null);
  const [selectedRoles, setSelectedRoles] = useState([]);
  const [roleError, setRoleError] = useState(null);
  const [isSubmittingRoles, setIsSubmittingRoles] = useState(false);


  // Modal Edit
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [editingUser, setEditingUser] = useState(null);
  const [editData, setEditData] = useState(initialEditForm);
  const [editError, setEditError] = useState(null);
  const [isSubmittingEdit, setIsSubmittingEdit] = useState(false);

  // Modal Reset Password
  const [isResetOpen, setIsResetOpen] = useState(false);
  const [resetTargetUser, setResetTargetUser] = useState(null);
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [resetError, setResetError] = useState(null);
  const [isSubmittingReset, setIsSubmittingReset] = useState(false);

  // Modal Delete
  const [deleteConfirmId, setDeleteConfirmId] = useState(null);
  const [deleteError, setDeleteError] = useState(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const fetchData = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const [usersData, brandsData, rolesData] = await Promise.all([
        listAdminUsers(),
        listBrands(),
        listRoles()
      ]);
      setUsers(usersData || []);
      setBrands(brandsData || []);
      setRoles(rolesData || []);
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal memuat data pengguna.";
      setErrorMessage(msg);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  // Pesan hasil dari halaman Tambah User (/users/new).
  useEffect(() => {
    if (location.state?.successMessage) setSuccessMessage(location.state.successMessage);
    if (location.state?.errorMessage) setErrorMessage(location.state.errorMessage);
    if (location.state) window.history.replaceState({}, '');
  }, [location.state]);


  // ─── Edit Handlers ──────────────────────────────────────────────────────────
  const handleOpenEdit = (user) => {
    setEditingUser(user);
    setEditData({
      email: user.email,
      role: user.brand_id === null ? 'super_admin' : 'travel_admin',
      brand_id: user.brand_id ? String(user.brand_id) : ''
    });
    setEditError(null);
    setIsEditOpen(true);
  };

  const handleCloseEdit = () => {
    if (!isSubmittingEdit) {
      setIsEditOpen(false);
      setEditingUser(null);
      setEditError(null);
    }
  };

  const handleSubmitEdit = async (e) => {
    e.preventDefault();
    setEditError(null);

    const email = editData.email.trim();
    if (!email) {
      setEditError("Email wajib diisi.");
      return;
    }

    if (editData.role === 'travel_admin' && !editData.brand_id) {
      setEditError("Brand wajib dipilih untuk role Admin Travel.");
      return;
    }

    setIsSubmittingEdit(true);
    try {
      const payload = {
        email,
        brand_id: editData.role === 'super_admin' ? null : Number(editData.brand_id)
      };

      await updateAdminUser(editingUser.id, payload);
      setSuccessMessage(`User "${email}" berhasil diperbarui.`);
      setIsEditOpen(false);
      setEditingUser(null);
      fetchData();
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal memperbarui user.";
      setEditError(msg);
    } finally {
      setIsSubmittingEdit(false);
    }
  };

  // ─── Reset Password Handlers ────────────────────────────────────────────────
  const handleOpenReset = (user) => {
    setResetTargetUser(user);
    setNewPassword('');
    setConfirmPassword('');
    setResetError(null);
    setIsResetOpen(true);
  };

  const handleCloseReset = () => {
    if (!isSubmittingReset) {
      setIsResetOpen(false);
      setResetTargetUser(null);
      setResetError(null);
    }
  };

  const handleSubmitReset = async (e) => {
    e.preventDefault();
    setResetError(null);

    if (newPassword.length < 8) {
      setResetError("Password baru minimal 8 karakter.");
      return;
    }

    if (newPassword !== confirmPassword) {
      setResetError("Konfirmasi password tidak cocok dengan password baru.");
      return;
    }

    setIsSubmittingReset(true);
    try {
      await resetAdminUserPassword(resetTargetUser.id, newPassword);
      setSuccessMessage(`Password untuk user "${resetTargetUser.email}" berhasil diubah.`);
      setIsResetOpen(false);
      setResetTargetUser(null);
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal mereset password user.";
      setResetError(msg);
    } finally {
      setIsSubmittingReset(false);
    }
  };

  // ─── Delete Handlers ────────────────────────────────────────────────────────
  const handleConfirmDelete = async () => {
    setIsDeleting(true);
    setDeleteError(null);
    try {
      await deleteAdminUser(deleteConfirmId);
      setSuccessMessage("User berhasil dihapus.");
      setDeleteConfirmId(null);
      fetchData();
    } catch (error) {
      const msg = error.response?.data?.error || "Gagal menghapus user.";
      setDeleteError(msg);
    } finally {
      setIsDeleting(false);
    }
  };

  // ─── Kelola Role (RBAC) ─────────────────────────────────────────────────────
  const roleNameOf = (slug) => roles.find(r => r.slug === slug)?.name || slug;

  const handleOpenRoles = (user) => {
    setRoleTargetUser(user);
    setSelectedRoles(user.roles || []);
    setRoleError(null);
  };

  const handleToggleRole = (slug) => {
    setSelectedRoles(prev =>
      prev.includes(slug) ? prev.filter(s => s !== slug) : [...prev, slug]
    );
  };

  const handleSubmitRoles = async (e) => {
    e.preventDefault();
    if (selectedRoles.length === 0) {
      setRoleError('Minimal satu role wajib dipilih.');
      return;
    }
    setIsSubmittingRoles(true);
    setRoleError(null);
    try {
      await setUserRoles(roleTargetUser.id, selectedRoles);
      setSuccessMessage(`Role untuk "${roleTargetUser.email}" berhasil diperbarui. Sesi aktif user tersebut dicabut — ia perlu login ulang.`);
      setRoleTargetUser(null);
      fetchData();
    } catch (error) {
      setRoleError(error.response?.data?.error || 'Gagal mengubah role user.');
    } finally {
      setIsSubmittingRoles(false);
    }
  };

  const userToDelete = users.find(u => u.id === deleteConfirmId);

  const totalSuperAdmin = users.filter(u => u.brand_id === null || u.brand_id === undefined).length;

  const columns = [
    { header: 'Email', key: 'email' },
    { header: 'Akses Brand', key: 'role' },
    { header: 'Role (RBAC)', key: 'rbac_roles' },
    { header: 'Aksi', key: 'aksi' }
  ];

  return (
    <div className="space-y-6">
      <PageHeader
        title="User Management"
        actionLabel="+ Tambah User"
        onAction={() => navigate('/users/new')}
      />

      {errorMessage && (
        <Alert variant="error">{errorMessage}</Alert>
      )}
      {successMessage && (
        <Alert variant="success">{successMessage}</Alert>
      )}

      {!isLoading && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatTile icon={Users} variant="primary" value={users.length} label="Total User Internal" />
          <StatTile icon={ShieldCheck} variant="violet" value={totalSuperAdmin} label="Super Admin Holding" />
          <StatTile icon={Building2} variant="info" value={users.length - totalSuperAdmin} label="Admin Brand" />
          <StatTile icon={KeyRound} variant="success" value={roles.length} label="Role Tersedia" />
        </div>
      )}

      {isLoading ? (
        <div className="flex justify-center p-8 bg-pure-white rounded-lg border border-neutral-200 shadow-sm">
          <LoadingSpinner />
        </div>
      ) : (
        <DataTable
          columns={columns}
          data={users}
          itemsPerPage={10}
          emptyMessage="Belum ada user terdaftar."
          searchPlaceholder="Cari user..."
          renderCell={(row, key) => {
            if (key === 'email') {
              const isSelf = currentUserId && Number(row.id) === Number(currentUserId);
              return (
                <div className="flex items-center gap-2">
                  <span className="font-heading font-medium text-neutral-900 text-sm">
                    {row.email}
                  </span>
                  {isSelf && (
                    <span className="text-[11px] font-medium font-body px-1.5 py-0.5 rounded-md bg-primary-50 text-primary-600 border border-primary-100">
                      Anda
                    </span>
                  )}
                </div>
              );
            }
            if (key === 'role') {
              const isSuperAdmin = row.brand_id === null || row.brand_id === undefined;
              const dotColor = isSuperAdmin
                ? '#F26522'
                : (row.brand_color || brands.find(b => String(b.id) === String(row.brand_id))?.primary_color || '#9CA3AF');
              // Chip lembut ala referensi: dot berwarna + teks gelap, bukan pill solid.
              return (
                <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[12px] font-medium font-body bg-white border border-neutral-200 text-neutral-700">
                  <span className="w-2 h-2 rounded-full shrink-0" style={{ backgroundColor: dotColor }} />
                  {getBrandAccessLabel(row)}
                </span>
              );
            }
            if (key === 'rbac_roles') {
              const userRoles = row.roles || [];
              if (userRoles.length === 0) {
                return <span className="text-xs text-neutral-400 font-body italic">Belum ada role</span>;
              }
              return (
                <div className="flex flex-wrap gap-1.5 max-w-xs">
                  {userRoles.map(slug => (
                    <span
                      key={slug}
                      className="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium font-body bg-primary-50 text-primary-700 border border-primary-200"
                      title={roleNameOf(slug)}
                    >
                      {roleNameOf(slug)}
                    </span>
                  ))}
                </div>
              );
            }
            if (key === 'aksi') {
              const isSelf = currentUserId && Number(row.id) === Number(currentUserId);

              // Tombol aksi kotak berbingkai 30px ala referensi.
              const actionBtn = "w-[30px] h-[30px] flex items-center justify-center rounded-lg border border-neutral-200 bg-white text-neutral-500 transition-colors";
              return (
                <div className="flex gap-1.5 items-center">
                  {/* Kelola Role */}
                  <button
                    onClick={() => handleOpenRoles(row)}
                    title="Kelola Role"
                    className={`${actionBtn} hover:text-primary-600 hover:bg-primary-50 hover:border-primary-200`}
                  >
                    <ShieldCheck className="w-4 h-4" strokeWidth={1.75} />
                  </button>

                  {/* Edit */}
                  <button
                    onClick={() => handleOpenEdit(row)}
                    title="Edit User"
                    className={`${actionBtn} hover:text-neutral-800 hover:bg-neutral-50`}
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.75" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                    </svg>
                  </button>

                  {/* Reset Password */}
                  <button
                    onClick={() => handleOpenReset(row)}
                    title="Reset Password"
                    className={`${actionBtn} hover:text-primary-600 hover:bg-primary-50 hover:border-primary-200`}
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.75" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
                    </svg>
                  </button>

                  {/* Hapus */}
                  {isSelf ? (
                    <button
                      disabled
                      title="Tidak bisa hapus akun sendiri"
                      className={`${actionBtn} opacity-40 cursor-not-allowed`}
                    >
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.75" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  ) : (
                    <button
                      onClick={() => {
                        setDeleteConfirmId(row.id);
                        setDeleteError(null);
                      }}
                      title="Hapus User"
                      className={`${actionBtn} hover:text-danger-600 hover:bg-danger-50 hover:border-danger-200`}
                    >
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.75" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  )}
                </div>
              );
            }
            return row[key];
          }}
        />
      )}

      {/* Modal Edit User */}
      <Modal
        isOpen={isEditOpen}
        onClose={handleCloseEdit}
        title="Edit User"
        size="md"
      >
        <form onSubmit={handleSubmitEdit} className="space-y-4">
          {editError && (
            <Alert variant="error">{editError}</Alert>
          )}

          <FormField label="Email Akun" required>
            <Input
              type="email"
              name="email"
              value={editData.email}
              onChange={(e) => setEditData(prev => ({ ...prev, email: e.target.value }))}
              required
            />
          </FormField>

          <CustomDropdown
            label="Akses Brand"
            required
            value={editData.role}
            onChange={(val) => setEditData(prev => ({ ...prev, role: val, brand_id: '' }))}
            options={roleOptions}
          />

          {editData.role === 'travel_admin' && (
            <CustomDropdown
              label="Pilih Brand"
              required
              value={editData.brand_id}
              onChange={(val) => setEditData(prev => ({ ...prev, brand_id: val }))}
              placeholder="Pilih Brand..."
              options={[
                { value: '', label: 'Pilih Brand...' },
                ...brands.map(b => ({ value: String(b.id), label: b.name }))
              ]}
            />
          )}

          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button
              type="button"
              variant="secondary"
              onClick={handleCloseEdit}
              disabled={isSubmittingEdit}
            >
              Batal
            </Button>
            <Button
              type="submit"
              variant="primary"
              disabled={isSubmittingEdit}
            >
              {isSubmittingEdit ? "Menyimpan..." : "Simpan Perubahan"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Modal Reset Password */}
      <Modal
        isOpen={isResetOpen}
        onClose={handleCloseReset}
        title={`Reset Password — ${resetTargetUser?.email || ''}`}
        size="md"
      >
        <form onSubmit={handleSubmitReset} className="space-y-4">
          {resetError && (
            <Alert variant="error">{resetError}</Alert>
          )}

          <p className="text-xs text-neutral-500 font-body">
            Masukkan password baru untuk akun pengguna ini. Pengguna dapat login menggunakan password baru setelah berhasil disimpan.
          </p>

          <FormField label="Password Baru" hint="Minimal 8 karakter." required>
            <Input
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              placeholder="••••••••"
              autoFocus
              required
            />
          </FormField>

          <FormField label="Konfirmasi Password Baru" required>
            <Input
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              placeholder="••••••••"
              required
            />
          </FormField>

          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button
              type="button"
              variant="secondary"
              onClick={handleCloseReset}
              disabled={isSubmittingReset}
            >
              Batal
            </Button>
            <Button
              type="submit"
              variant="primary"
              disabled={isSubmittingReset}
            >
              {isSubmittingReset ? "Menyimpan..." : "Ubah Password"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Modal Kelola Role (RBAC) */}
      <Modal
        isOpen={!!roleTargetUser}
        onClose={() => { if (!isSubmittingRoles) { setRoleTargetUser(null); setRoleError(null); } }}
        title={`Kelola Role — ${roleTargetUser?.email || ''}`}
        size="md"
      >
        <form onSubmit={handleSubmitRoles} className="space-y-4">
          {roleError && <Alert variant="error">{roleError}</Alert>}

          <p className="text-xs text-neutral-500 font-body">
            Satu user boleh memegang lebih dari satu role. Mengubah role akan mencabut sesi aktif user tersebut (wajib login ulang).
          </p>

          <div className="max-h-72 overflow-y-auto space-y-1.5 pr-1">
            {roles.map(role => (
              <label
                key={role.slug}
                className={`flex items-start gap-3 p-3 rounded-xl border cursor-pointer transition-colors ${
                  selectedRoles.includes(role.slug)
                    ? 'border-primary-300 bg-primary-50'
                    : 'border-neutral-200 bg-white hover:bg-neutral-50'
                }`}
              >
                <input
                  type="checkbox"
                  className="mt-0.5 h-4 w-4 rounded border-neutral-300 text-primary-500 focus:ring-primary-500 accent-[#F26522]"
                  checked={selectedRoles.includes(role.slug)}
                  onChange={() => handleToggleRole(role.slug)}
                />
                <span className="min-w-0">
                  <span className="block text-sm font-heading font-semibold text-neutral-900">{role.name}</span>
                  <span className="block text-[11px] font-body text-neutral-500">
                    {role.slug} · {(role.permissions || []).length} permission
                  </span>
                </span>
              </label>
            ))}
          </div>

          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button
              type="button"
              variant="secondary"
              onClick={() => { setRoleTargetUser(null); setRoleError(null); }}
              disabled={isSubmittingRoles}
            >
              Batal
            </Button>
            <Button type="submit" variant="primary" disabled={isSubmittingRoles}>
              {isSubmittingRoles ? 'Menyimpan...' : 'Simpan Role'}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Modal Konfirmasi Hapus */}
      <Modal
        isOpen={!!deleteConfirmId}
        onClose={() => {
          if (!isDeleting) {
            setDeleteConfirmId(null);
            setDeleteError(null);
          }
        }}
        title="Konfirmasi Hapus User"
      >
        <div className="space-y-4">
          {deleteError && (
            <Alert variant="error">{deleteError}</Alert>
          )}

          <p className="text-sm font-body text-neutral-700">
            Apakah Anda yakin ingin menghapus user <strong className="text-neutral-900 font-heading">{userToDelete?.email}</strong>?
          </p>
          <p className="text-xs font-body text-neutral-500">
            Tindakan ini tidak dapat dibatalkan. User yang dihapus tidak akan bisa login lagi ke sistem.
          </p>

          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                setDeleteConfirmId(null);
                setDeleteError(null);
              }}
              disabled={isDeleting}
            >
              Batal
            </Button>
            <Button
              type="button"
              variant="danger"
              onClick={handleConfirmDelete}
              disabled={isDeleting}
            >
              {isDeleting ? "Menghapus..." : "Hapus User"}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default UserManagementPage;
