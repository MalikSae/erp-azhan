import React, { useContext, useEffect, useMemo, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { Users, ShieldCheck, Building2, UserX, Pencil, KeyRound, Power } from 'lucide-react';
import PageHeader from 'shared/ui/PageHeader';
import DataTable from 'shared/ui/DataTable';
import Button from 'shared/ui/Button';
import Modal from 'shared/ui/Modal';
import Input from 'shared/ui/Input';
import Alert from 'shared/ui/Alert';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import StatTile from 'shared/ui/StatTile';
import ActionMenu from 'shared/ui/ActionMenu';
import { AuthContext } from '../context/AuthContext';
import { listAdminUsers, setAdminUserStatus, resetAdminUserPassword } from '../api/adminUsers';
import { listRoles } from '../api/roles';
import { errMsg, StatusChip, AksesChip } from '../components/penggunaUi';

const FILTERS = [
  { value: 'aktif', label: 'Aktif' },
  { value: 'nonaktif', label: 'Nonaktif' },
  { value: 'semua', label: 'Semua' },
];

// Daftar Pengguna (SPEK-PENGGUNA-ORGANISASI §6). Satu orang = satu Pengguna;
// pembuatan & perubahan lewat form halaman penuh (/users/new, /users/:id).
const PenggunaPage = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { currentUserId } = useContext(AuthContext);

  const [users, setUsers] = useState([]);
  const [roles, setRoles] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState(null);
  const [successMessage, setSuccessMessage] = useState(null);
  const [filter, setFilter] = useState('aktif');

  const [statusTarget, setStatusTarget] = useState(null);
  const [statusError, setStatusError] = useState(null);
  const [isSavingStatus, setIsSavingStatus] = useState(false);

  const [resetTarget, setResetTarget] = useState(null);
  const [resetForm, setResetForm] = useState({ password: '', confirm: '' });
  const [resetError, setResetError] = useState(null);
  const [isSavingReset, setIsSavingReset] = useState(false);

  const fetchData = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const [usersData, rolesData] = await Promise.all([listAdminUsers(), listRoles()]);
      setUsers(usersData || []);
      setRoles(rolesData || []);
    } catch (error) {
      setErrorMessage(errMsg(error, 'Gagal memuat data pengguna.'));
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => { fetchData(); }, []);

  // Pesan hasil dari form pengguna.
  useEffect(() => {
    if (location.state?.successMessage) setSuccessMessage(location.state.successMessage);
    if (location.state) window.history.replaceState({}, '');
  }, [location.state]);

  const roleName = (slug) => roles.find(r => r.slug === slug)?.name || slug;
  const isSelf = (row) => currentUserId && Number(row.id) === Number(currentUserId);

  const stats = useMemo(() => {
    const aktif = users.filter(u => u.is_active);
    return {
      aktif: aktif.length,
      holding: aktif.filter(u => u.brand_id === null || u.brand_id === undefined).length,
      brand: aktif.filter(u => u.brand_id !== null && u.brand_id !== undefined).length,
      nonaktif: users.length - aktif.length,
    };
  }, [users]);

  const rows = useMemo(() => users
    .filter(u => filter === 'semua' || (filter === 'aktif' ? u.is_active : !u.is_active))
    // Kolom bantu pencarian: nama role & unit ikut bisa dicari.
    .map(u => ({
      ...u,
      _cari: [...(u.roles || []).map(roleName), ...(u.keanggotaan || []).map(k => k.unit_nama)].join(' '),
    })), [users, filter, roles]);

  const handleStatus = async () => {
    setIsSavingStatus(true);
    setStatusError(null);
    try {
      const next = !statusTarget.is_active;
      await setAdminUserStatus(statusTarget.id, next);
      setSuccessMessage(next
        ? `Pengguna "${statusTarget.display_name}" diaktifkan kembali.`
        : `Pengguna "${statusTarget.display_name}" dinonaktifkan. Semua sesinya dicabut.`);
      setStatusTarget(null);
      fetchData();
    } catch (error) {
      setStatusError(errMsg(error, 'Gagal mengubah status pengguna.'));
    } finally {
      setIsSavingStatus(false);
    }
  };

  const handleReset = async (e) => {
    e.preventDefault();
    setResetError(null);
    if (resetForm.password.length < 8) { setResetError('Password baru minimal 8 karakter.'); return; }
    if (resetForm.password !== resetForm.confirm) { setResetError('Konfirmasi password tidak cocok.'); return; }
    setIsSavingReset(true);
    try {
      await resetAdminUserPassword(resetTarget.id, resetForm.password);
      setSuccessMessage(`Password "${resetTarget.display_name}" berhasil diganti. Sesi lamanya dicabut.`);
      setResetTarget(null);
    } catch (error) {
      setResetError(errMsg(error, 'Gagal mengganti password.'));
    } finally {
      setIsSavingReset(false);
    }
  };

  const columns = [
    { header: 'Nama', key: 'display_name' },
    { header: 'Akses Brand', key: 'akses' },
    { header: 'Role', key: 'roles' },
    { header: 'Tim / Unit', key: 'keanggotaan' },
    { header: 'Status', key: 'status' },
    { header: 'Aksi', key: 'aksi' },
  ];

  return (
    <div className="space-y-6">
      <PageHeader title="Pengguna" actionLabel="+ Tambah Pengguna" onAction={() => navigate('/users/new')} />

      {errorMessage && <Alert variant="error" onClose={() => setErrorMessage(null)}>{errorMessage}</Alert>}
      {successMessage && <Alert variant="success" onClose={() => setSuccessMessage(null)}>{successMessage}</Alert>}

      {!isLoading && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatTile icon={Users} variant="primary" value={stats.aktif} label="Pengguna Aktif" />
          <StatTile icon={ShieldCheck} variant="violet" value={stats.holding} label="Akses Holding" />
          <StatTile icon={Building2} variant="info" value={stats.brand} label="Akses Brand" />
          <StatTile icon={UserX} variant="danger" value={stats.nonaktif} label="Nonaktif" />
        </div>
      )}

      {isLoading ? (
        <div className="flex justify-center p-8 bg-white rounded-2xl border border-neutral-200 shadow-card"><LoadingSpinner /></div>
      ) : (
        <DataTable
          columns={columns}
          data={rows}
          itemsPerPage={10}
          searchPlaceholder="Cari nama, email, role, atau tim..."
          emptyMessage={filter === 'nonaktif' ? 'Tidak ada pengguna nonaktif.' : 'Belum ada pengguna.'}
          toolbarActions={(
            <div className="inline-flex rounded-lg border border-neutral-200 bg-white p-0.5">
              {FILTERS.map(f => (
                <button
                  key={f.value}
                  type="button"
                  onClick={() => setFilter(f.value)}
                  className={`px-3 py-1.5 text-xs font-medium font-body rounded-md transition-colors ${
                    filter === f.value ? 'bg-primary-500 text-white' : 'text-neutral-600 hover:bg-neutral-50'
                  }`}
                >
                  {f.label}
                </button>
              ))}
            </div>
          )}
          renderCell={(row, key) => {
            if (key === 'display_name') {
              return (
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <button type="button" onClick={() => navigate(`/users/${row.id}`)} className="font-heading font-semibold text-neutral-900 hover:text-primary-600 text-left">
                      {row.display_name}
                    </button>
                    {isSelf(row) && <span className="text-[11px] font-body text-neutral-400">(Anda)</span>}
                  </div>
                  <div className="text-[11px] font-body text-neutral-400 truncate">
                    {row.jabatan ? `${row.jabatan} · ` : ''}{row.email}
                  </div>
                </div>
              );
            }
            if (key === 'akses') return <AksesChip brandId={row.brand_id} brandName={row.brand_name} brandColor={row.brand_color} />;
            if (key === 'roles') {
              const list = row.roles || [];
              if (list.length === 0) return <span className="text-xs text-neutral-400 font-body">—</span>;
              return (
                <span className="block max-w-[220px] text-[13px] font-body text-neutral-700">
                  {list.map(roleName).join(', ')}
                </span>
              );
            }
            if (key === 'keanggotaan') {
              const list = row.keanggotaan || [];
              if (list.length === 0) return <span className="text-xs text-neutral-400 font-body">—</span>;
              return (
                <div className="max-w-[240px] space-y-0.5 text-[13px] font-body text-neutral-700">
                  {list.map(k => (
                    <div key={k.org_unit_id}>
                      {k.unit_nama}
                      {k.posisi === 'pemimpin' && <span className="text-[11px] text-neutral-400"> · Pemimpin</span>}
                    </div>
                  ))}
                </div>
              );
            }
            if (key === 'status') return <StatusChip active={row.is_active} />;
            if (key === 'aksi') {
              const menuItems = [
                { label: 'Edit', icon: Pencil, tooltip: 'Edit Pengguna', onClick: () => navigate(`/users/${row.id}`) },
                {
                  label: 'Ganti Password',
                  icon: KeyRound,
                  disabled: !row.is_active,
                  disabledTooltip: 'Pengguna nonaktif',
                  onClick: () => { setResetTarget(row); setResetForm({ password: '', confirm: '' }); setResetError(null); },
                },
                {
                  label: row.is_active ? 'Nonaktifkan' : 'Aktifkan',
                  icon: Power,
                  danger: row.is_active,
                  disabled: isSelf(row),
                  disabledTooltip: 'Tidak bisa menonaktifkan akun sendiri',
                  onClick: () => { setStatusTarget(row); setStatusError(null); },
                },
              ];
              return <ActionMenu items={menuItems} align="right" />;
            }
            return row[key];
          }}
        />
      )}

      {/* Konfirmasi nonaktif / aktif kembali */}
      <Modal
        isOpen={!!statusTarget}
        onClose={() => { if (!isSavingStatus) setStatusTarget(null); }}
        title={statusTarget?.is_active ? 'Nonaktifkan Pengguna' : 'Aktifkan Kembali Pengguna'}
      >
        <div className="space-y-4">
          {statusError && <Alert variant="error">{statusError}</Alert>}
          {statusTarget?.is_active ? (
            <p className="text-sm font-body text-neutral-700">
              <strong className="font-heading text-neutral-900">{statusTarget?.display_name}</strong> tidak akan bisa login lagi dan semua sesinya langsung dicabut.
              Riwayat transaksinya tetap utuh, dan akun bisa diaktifkan kembali kapan saja.
            </p>
          ) : (
            <p className="text-sm font-body text-neutral-700">
              <strong className="font-heading text-neutral-900">{statusTarget?.display_name}</strong> akan bisa login kembali dengan role dan akses yang tersimpan.
            </p>
          )}
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button type="button" variant="secondary" onClick={() => setStatusTarget(null)} disabled={isSavingStatus}>Batal</Button>
            <Button type="button" variant={statusTarget?.is_active ? 'danger' : 'primary'} onClick={handleStatus} disabled={isSavingStatus}>
              {isSavingStatus ? 'Menyimpan...' : (statusTarget?.is_active ? 'Nonaktifkan' : 'Aktifkan Kembali')}
            </Button>
          </div>
        </div>
      </Modal>

      {/* Ganti password */}
      <Modal
        isOpen={!!resetTarget}
        onClose={() => { if (!isSavingReset) setResetTarget(null); }}
        title={`Ganti Password — ${resetTarget?.display_name || ''}`}
      >
        <form onSubmit={handleReset} className="space-y-1">
          {resetError && <Alert variant="error" className="mb-4">{resetError}</Alert>}
          <Input label="Password Baru" type="password" required value={resetForm.password}
            onChange={(e) => setResetForm(p => ({ ...p, password: e.target.value }))} autoFocus />
          <Input label="Konfirmasi Password Baru" type="password" required value={resetForm.confirm}
            onChange={(e) => setResetForm(p => ({ ...p, confirm: e.target.value }))} />
          <p className="text-xs font-body text-neutral-500 pb-2">Semua sesi pengguna ini akan dicabut; ia perlu login ulang dengan password baru.</p>
          <div className="flex justify-end gap-3 pt-4 border-t border-neutral-200">
            <Button type="button" variant="secondary" onClick={() => setResetTarget(null)} disabled={isSavingReset}>Batal</Button>
            <Button type="submit" variant="primary" disabled={isSavingReset}>{isSavingReset ? 'Menyimpan...' : 'Simpan Password'}</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
};

export default PenggunaPage;
