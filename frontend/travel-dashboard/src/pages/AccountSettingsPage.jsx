import { useState } from 'react';
import { KeyRound, LockKeyhole, ShieldCheck } from 'lucide-react';
import api from '../api/client';
import Alert from '../components/ui/Alert';
import Button from '../components/ui/Button';
import Card from '../components/ui/Card';
import Input from '../components/ui/Input';
import PageHeader from '../components/ui/PageHeader';
import { useAuth } from '../context/AuthContext';

const initialForm = { currentPassword: '', newPassword: '', confirmPassword: '' };

export default function AccountSettingsPage() {
  const { user, brandInfo } = useAuth();
  const [form, setForm] = useState(initialForm);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const updateField = (field) => (event) => {
    setForm((previous) => ({ ...previous, [field]: event.target.value }));
    setError('');
    setSuccess('');
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    setError('');
    setSuccess('');

    if (form.newPassword.length < 8) {
      setError('Password baru minimal 8 karakter.');
      return;
    }
    if (form.newPassword !== form.confirmPassword) {
      setError('Konfirmasi password baru tidak cocok.');
      return;
    }

    setSaving(true);
    try {
      await api.put('/api/admin/account/password', {
        current_password: form.currentPassword,
        new_password: form.newPassword,
      });
      setForm(initialForm);
      setSuccess('Password berhasil diubah. Gunakan password baru pada login berikutnya.');
    } catch (requestError) {
      setError(requestError.response?.data?.error || 'Password gagal diubah. Silakan coba lagi.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Pengaturan akun"
        subtitle="Kelola keamanan akun Admin Travel Anda."
      />

      {(error || success) && (
        <Alert variant={error ? 'error' : 'success'} message={error || success} />
      )}

      <div className="grid gap-5 lg:grid-cols-[minmax(0,1.35fr)_minmax(260px,0.65fr)]">
        <Card className="p-0 overflow-hidden">
          <div className="border-b border-neutral-100 px-5 py-5 md:px-6">
            <div className="flex items-start gap-3">
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary-50 text-primary-700">
                <KeyRound size={19} />
              </div>
              <div>
                <h2 className="font-heading text-base font-semibold text-neutral-900">Informasi akun</h2>
                <p className="mt-1 text-sm text-neutral-500">Data akun dikelola oleh Admin Master dan tidak dapat diubah di sini.</p>
              </div>
            </div>
          </div>

          <form onSubmit={handleSubmit} className="space-y-5 px-5 py-5 md:px-6 md:py-6">
            <div className="grid gap-4 md:grid-cols-2">
              <Input label="Nama akun" value={user?.display_name || 'Admin Travel'} disabled readOnly />
              <Input label="Email akun" value={user?.email || ''} disabled readOnly />
              <Input label="Brand" value={brandInfo?.name || ''} disabled readOnly />
              <Input label="Peran" value="Admin Travel" disabled readOnly />
            </div>

            <div className="border-t border-neutral-100 pt-5">
              <div className="mb-4 flex items-center gap-2">
                <LockKeyhole size={16} className="text-neutral-500" />
                <h3 className="font-heading text-sm font-semibold text-neutral-900">Ubah password</h3>
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <Input
                  label="Password saat ini"
                  type="password"
                  value={form.currentPassword}
                  onChange={updateField('currentPassword')}
                  placeholder="Masukkan password saat ini"
                  required
                  className="md:col-span-2"
                />
                <Input
                  label="Password baru"
                  type="password"
                  value={form.newPassword}
                  onChange={updateField('newPassword')}
                  placeholder="Minimal 8 karakter"
                  required
                />
                <Input
                  label="Konfirmasi password baru"
                  type="password"
                  value={form.confirmPassword}
                  onChange={updateField('confirmPassword')}
                  placeholder="Ulangi password baru"
                  required
                />
              </div>
            </div>

            <div className="flex justify-end border-t border-neutral-100 pt-5">
              <Button type="submit" isLoading={saving} disabled={saving} icon={<LockKeyhole size={15} />}>
                Simpan password
              </Button>
            </div>
          </form>
        </Card>

        <Card className="h-fit bg-neutral-50/70">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-success-50 text-success-700">
            <ShieldCheck size={19} />
          </div>
          <h2 className="mt-4 font-heading text-base font-semibold text-neutral-900">Akses aman</h2>
          <p className="mt-2 text-sm leading-6 text-neutral-500">
            Email, brand, dan peran akun sengaja dikunci agar perubahan identitas hanya dilakukan oleh Admin Master.
          </p>
          <ul className="mt-5 space-y-3 text-sm text-neutral-600">
            <li className="flex gap-2"><span className="mt-2 h-1.5 w-1.5 shrink-0 rounded-full bg-success-500" />Password baru minimal 8 karakter.</li>
            <li className="flex gap-2"><span className="mt-2 h-1.5 w-1.5 shrink-0 rounded-full bg-success-500" />Password saat ini wajib diverifikasi.</li>
            <li className="flex gap-2"><span className="mt-2 h-1.5 w-1.5 shrink-0 rounded-full bg-success-500" />Perubahan berlaku pada login berikutnya.</li>
          </ul>
        </Card>
      </div>
    </div>
  );
}
