import React, { useContext, useEffect, useMemo, useState } from 'react';
import { passwordTerlaluPanjang } from 'shared';
import { useNavigate, useParams } from 'react-router-dom';
import PageHeader from 'shared/ui/PageHeader';
import FormLayout, { FormSection, SidebarSection, FormAside } from 'shared/ui/FormLayout';
import Button from 'shared/ui/Button';
import Alert from 'shared/ui/Alert';
import Input from 'shared/ui/Input';
import Modal from 'shared/ui/Modal';
import Checkbox from 'shared/ui/Checkbox';
import TextButton from 'shared/ui/TextButton';
import { Check } from 'lucide-react';
import CustomDropdown from 'shared/ui/CustomDropdown';
import LoadingSpinner from 'shared/ui/LoadingSpinner';
import { AuthContext } from '../context/AuthContext';
import {
  getAdminUser, createAdminUser, updateAdminUser,
  setAdminUserStatus, resetAdminUserPassword, deleteAdminUser,
} from '../api/adminUsers';
import { listBrandOptions } from '../api/brands';
import { errMsg, StatusChip } from '../components/penggunaUi';
import {
  SUPER_ADMIN, TIM, AKSES_LABEL, BARIS_IZIN, IZIN_KHUSUS, TEMPLAT_AKSES, kodeBoleh, rapikanIzin,
} from '../constants/izinAkses';

const emptyForm = {
  display_name: '', email: '', no_hp: '',
  cakupan: 'holding', brand_id: '', akses: TIM, password: '',
};


// Form Pengguna (SPEK-AKSES-PENGGUNA-2026-11-25 §6): profil, akses, dan izin
// disimpan atomik dalam satu permintaan.
const PenggunaFormPage = () => {
  const { id } = useParams();
  const isEdit = Boolean(id);
  const navigate = useNavigate();
  const { currentUserId } = useContext(AuthContext);
  const isSelf = isEdit && currentUserId && Number(id) === Number(currentUserId);

  const [brands, setBrands] = useState([]);
  const [user, setUser] = useState(null); // data asli (mode edit)
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState(null);

  const [form, setForm] = useState(emptyForm);
  const [izin, setIzin] = useState(() => new Set());
  const [templat, setTemplat] = useState('');
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
        const [brandsData, userData] = await Promise.all([
          listBrandOptions(), isEdit ? getAdminUser(id) : Promise.resolve(null),
        ]);
        if (!active) return;
        setBrands(brandsData || []);
        if (userData) {
          setUser(userData);
          const holding = userData.brand_id === null || userData.brand_id === undefined;
          setForm({
            display_name: userData.display_name || '',
            email: userData.email || '',
            no_hp: userData.no_hp || '',
            cakupan: holding ? 'holding' : 'brand',
            brand_id: holding ? '' : String(userData.brand_id),
            akses: userData.akses === SUPER_ADMIN ? SUPER_ADMIN : TIM,
            password: '',
          });
          setIzin(new Set(userData.permissions || []));
        }
      } catch (error) {
        if (active) setLoadError(errMsg(error, 'Data gagal dimuat. Coba lagi.'));
      } finally {
        if (active) setIsLoading(false);
      }
    };
    load();
    return () => { active = false; };
  }, [id, isEdit]);

  const isBrand = form.cakupan === 'brand';
  const isTim = form.akses === TIM;

  // Baris dan izin khusus yang boleh dipilih untuk akses ini; sisanya disembunyikan.
  const baris = useMemo(() => BARIS_IZIN
    .map((b) => ({
      ...b,
      bisaLihat: b.lihat.length > 0 && (!isBrand || b.brand.lihat),
      bisaKelola: b.kelola.length > 0 && (!isBrand || b.brand.kelola),
    }))
    .filter((b) => b.bisaLihat || b.bisaKelola), [isBrand]);
  const khusus = useMemo(() => IZIN_KHUSUS.filter((k) => !isBrand || k.brand), [isBrand]);
  const pilihanTemplat = useMemo(() => TEMPLAT_AKSES.filter((t) => !isBrand || t.brand), [isBrand]);

  const punya = (kode) => kode.some((k) => izin.has(k));
  const ubah = (tambah, hapus = []) => {
    setTemplat('');
    setIzin((prev) => {
      const next = new Set(prev);
      hapus.forEach((k) => next.delete(k));
      tambah.forEach((k) => next.add(k));
      return next;
    });
  };
  // Kelola menyertakan Lihat; Lihat tidak bisa dilepas selama Kelola aktif.
  const setLihat = (b, on) => (on ? ubah(b.lihat) : ubah([], b.lihat));
  const setKelola = (b, on) => (on ? ubah([...b.kelola, ...(b.bisaLihat ? b.lihat : [])]) : ubah([], b.kelola));

  const ubahCakupan = (cakupan) => {
    if (cakupan === form.cakupan) return;
    const boleh = kodeBoleh(cakupan);
    setForm((p) => ({ ...p, cakupan, brand_id: '', akses: cakupan === 'brand' ? TIM : p.akses }));
    setIzin((prev) => new Set([...prev].filter((k) => boleh.has(k))));
    setTemplat('');
  };

  const pakaiTemplat = (idTemplat) => {
    const t = TEMPLAT_AKSES.find((x) => x.id === idTemplat);
    if (!t) return;
    const boleh = kodeBoleh(form.cakupan);
    setIzin(new Set(t.kode.filter((k) => boleh.has(k))));
    setTemplat(idTemplat);
  };

  // ─── Simpan ──────────────────────────────────────────────────────────────
  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitError(null);
    const permissions = isTim ? rapikanIzin([...izin], form.cakupan) : [];
    if (isBrand && !form.brand_id) { setSubmitError('Pilih brand.'); return; }
    if (isTim && permissions.length === 0) { setSubmitError('Pilih minimal satu izin.'); return; }
    if (!isEdit && form.password.trim().length < 8) { setSubmitError('Kata sandi minimal 8 karakter.'); return; }
    if (!isEdit && passwordTerlaluPanjang(form.password.trim())) { setSubmitError('Kata sandi terlalu panjang.'); return; }

    const payload = {
      display_name: form.display_name.trim(),
      email: form.email.trim(),
      no_hp: form.no_hp.trim() || null,
      brand_id: isBrand ? Number(form.brand_id) : null,
      akses: form.akses,
      permissions,
    };

    setIsSubmitting(true);
    try {
      if (isEdit) {
        await updateAdminUser(id, payload);
        navigate('/users', { state: { successMessage: 'Perubahan disimpan.' } });
      } else {
        await createAdminUser({ ...payload, password: form.password.trim() });
        navigate('/users', { state: { successMessage: 'Pengguna ditambahkan.' } });
      }
    } catch (error) {
      setSubmitError(errMsg(error, 'Pengguna gagal disimpan. Coba lagi.'));
      document.querySelector('main')?.scrollTo?.({ top: 0, behavior: 'smooth' });
    } finally {
      setIsSubmitting(false);
    }
  };

  // ─── Tindakan akun (edit) ────────────────────────────────────────────────
  const bukaAksi = (jenis) => { setActionModal(jenis); setActionError(null); setNewPassword(''); };
  const runAction = async () => {
    setIsActing(true);
    setActionError(null);
    try {
      if (actionModal === 'status') {
        const updated = await setAdminUserStatus(id, !user.is_active);
        setUser(updated);
        setActionModal(null);
      } else if (actionModal === 'delete') {
        await deleteAdminUser(id);
        navigate('/users', { state: { successMessage: 'Pengguna dihapus.' } });
      } else if (actionModal === 'password') {
        if (newPassword.trim().length < 8) { setActionError('Kata sandi minimal 8 karakter.'); return; }
        if (passwordTerlaluPanjang(newPassword.trim())) { setActionError('Kata sandi terlalu panjang.'); return; }
        await resetAdminUserPassword(id, newPassword.trim());
        setActionModal(null);
      }
    } catch (error) {
      setActionError(errMsg(error, 'Aksi gagal. Coba lagi.'));
    } finally {
      setIsActing(false);
    }
  };

  if (isLoading) return <LoadingSpinner variant="card" />;

  const judulModal = actionModal === 'delete' ? 'Hapus Pengguna'
    : actionModal === 'password' ? 'Ganti Kata Sandi'
      : (user?.is_active ? 'Nonaktifkan Pengguna' : 'Aktifkan Pengguna');
  const labelAksi = actionModal === 'delete' ? 'Hapus'
    : actionModal === 'password' ? 'Simpan'
      : (user?.is_active ? 'Nonaktifkan' : 'Aktifkan');
  const aksiMerusak = actionModal === 'delete' || (actionModal === 'status' && user?.is_active);

  return (
    <div className="space-y-6">
      <PageHeader title={isEdit ? 'Edit Pengguna' : 'Tambah Pengguna'} onBack={() => navigate('/users')} />

      {loadError && <Alert variant="error">{loadError}</Alert>}
      {submitError && <Alert variant="error" onClose={() => setSubmitError(null)}>{submitError}</Alert>}

      <form onSubmit={handleSubmit}>
        <FormLayout>
          <FormSection id="sec-profil" title="Profil">
            <div className="grid grid-cols-1 gap-x-4 md:grid-cols-2">
              <Input label="Nama" required value={form.display_name} placeholder="Ahmad Fauzi"
                onChange={(e) => setForm((p) => ({ ...p, display_name: e.target.value }))} autoFocus={!isEdit} />
              <Input label="Email" type="email" required value={form.email} placeholder="ahmad@azhan.id"
                onChange={(e) => setForm((p) => ({ ...p, email: e.target.value }))} />
              <Input label="No. HP" value={form.no_hp} placeholder="081234567890"
                onChange={(e) => setForm((p) => ({ ...p, no_hp: e.target.value }))} />
              {!isEdit && (
                <Input label="Kata sandi" type="password" required value={form.password}
                  onChange={(e) => setForm((p) => ({ ...p, password: e.target.value }))} />
              )}
            </div>
          </FormSection>

          <FormSection id="sec-akses" title="Akses">
            <div className="grid grid-cols-1 gap-x-4 md:grid-cols-2">
              <CustomDropdown
                label="Cakupan data"
                required
                value={form.cakupan}
                onChange={ubahCakupan}
                options={[
                  { value: 'holding', label: 'Holding (semua brand)' },
                  { value: 'brand', label: 'Satu brand' },
                ]}
              />
              {isBrand ? (
                <CustomDropdown
                  label="Brand"
                  required
                  value={form.brand_id}
                  onChange={(val) => setForm((p) => ({ ...p, brand_id: val }))}
                  placeholder="Pilih brand"
                  options={brands.map((b) => ({ value: String(b.id), label: b.name }))}
                />
              ) : (
                <CustomDropdown
                  label="Jenis akses"
                  required
                  value={form.akses}
                  onChange={(val) => setForm((p) => ({ ...p, akses: val }))}
                  options={[
                    { value: TIM, label: AKSES_LABEL[TIM] },
                    { value: SUPER_ADMIN, label: AKSES_LABEL[SUPER_ADMIN] },
                  ]}
                />
              )}
            </div>
          </FormSection>

          {isTim && (
            <FormSection
              id="sec-izin"
              title="Izin"
              aside={(
                <div className="w-44">
                  <CustomDropdown
                    value={templat}
                    onChange={pakaiTemplat}
                    placeholder="Isi dari templat"
                    className="!mb-0"
                    options={pilihanTemplat.map((t) => ({ value: t.id, label: t.label }))}
                  />
                </div>
              )}
            >
              <div className="grid grid-cols-[minmax(0,1fr)_4rem_4rem] items-center border-b border-neutral-200 pb-2 font-body text-xs font-medium text-neutral-500">
                <span>Menu</span>
                <span className="text-center">Lihat</span>
                <span className="text-center">Kelola</span>
              </div>
              <ul className="divide-y divide-neutral-100">
                {baris.map((b) => {
                  const kelola = b.bisaKelola && punya(b.kelola);
                  const lihat = b.bisaLihat && (kelola || punya(b.lihat));
                  return (
                    <li key={b.id} className="grid grid-cols-[minmax(0,1fr)_4rem_4rem] items-center py-2.5">
                      <span className="font-body text-sm text-neutral-800">{b.label}</span>
                      <span className="flex justify-center">
                        {/* Kelola sudah termasuk Lihat: tanda centang biasa, bukan kotak nonaktif. */}
                        {b.bisaLihat && (kelola ? (
                          <Check size={16} className="text-neutral-400" aria-label={`Lihat ${b.label} termasuk`} />
                        ) : (
                          <Checkbox ariaLabel={`Lihat ${b.label}`} checked={lihat}
                            onChange={(on) => setLihat(b, on)} />
                        ))}
                      </span>
                      <span className="flex justify-center">
                        {b.bisaKelola && (
                          <Checkbox ariaLabel={`Kelola ${b.label}`} checked={kelola}
                            onChange={(on) => setKelola(b, on)} />
                        )}
                      </span>
                    </li>
                  );
                })}
              </ul>
            </FormSection>
          )}

          {isTim && (
            <FormSection id="sec-khusus" title="Izin Khusus">
              <div className="grid grid-cols-1 gap-x-4 gap-y-3 md:grid-cols-2">
                {khusus.map((k) => (
                  <Checkbox key={k.id} label={k.label} checked={punya(k.kode)}
                    onChange={(on) => (on ? ubah(k.kode) : ubah([], k.kode))} />
                ))}
              </div>
            </FormSection>
          )}

          <FormAside>
            {isEdit && user && (
              <SidebarSection title="Akun">
                <div className="space-y-3">
                  <StatusChip active={user.is_active} />
                  <div className="flex flex-col items-start gap-2">
                    {user.is_active && (
                      <TextButton onClick={() => bukaAksi('password')}>Ganti kata sandi</TextButton>
                    )}
                    {!isSelf && (
                      <TextButton onClick={() => bukaAksi('status')}>
                        {user.is_active ? 'Nonaktifkan' : 'Aktifkan'}
                      </TextButton>
                    )}
                    {!isSelf && (
                      <TextButton onClick={() => bukaAksi('delete')}>Hapus</TextButton>
                    )}
                  </div>
                </div>
              </SidebarSection>
            )}
            <SidebarSection>
              <Button type="submit" variant="primary" isLoading={isSubmitting} disabled={Boolean(loadError)} className="w-full justify-center">
                Simpan
              </Button>
            </SidebarSection>
          </FormAside>
        </FormLayout>
      </form>

      {/* Modal tindakan akun: di luar form supaya Enter tidak menyimpan form pengguna. */}
      <Modal
        isOpen={!!actionModal}
        onClose={() => { if (!isActing) setActionModal(null); }}
        title={judulModal}
        size="sm"
        footer={(
          <>
            <Button type="button" variant="secondary" onClick={() => setActionModal(null)} disabled={isActing}>Batal</Button>
            <Button type="button" variant={aksiMerusak ? 'danger' : 'primary'} onClick={runAction} isLoading={isActing}>{labelAksi}</Button>
          </>
        )}
      >
        <div className="space-y-4">
          {actionError && <Alert variant="error">{actionError}</Alert>}
          {actionModal === 'password' && (
            <Input label="Kata sandi baru" type="password" required value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)} autoFocus />
          )}
          {actionModal === 'status' && (
            <p className="font-body text-sm text-neutral-700">
              {user?.is_active ? `Nonaktifkan ${user?.display_name}?` : `Aktifkan ${user?.display_name}?`}
            </p>
          )}
          {actionModal === 'delete' && (
            <p className="font-body text-sm text-neutral-700">Hapus {user?.display_name}?</p>
          )}
        </div>
      </Modal>
    </div>
  );
};

export default PenggunaFormPage;
