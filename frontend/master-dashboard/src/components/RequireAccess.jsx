import React, { useContext } from 'react';
import { Link } from 'react-router-dom';
import { ShieldAlert } from 'lucide-react';
import { AuthContext } from '../context/AuthContext';

// Penjaga halaman berdasarkan izin pengguna. Penegakan sebenarnya ada di API (izin + brand);
// ini hanya mencegah admin brand membuka halaman yang bukan haknya lewat alamat langsung.
//
//   <RequireAccess perm="brand.view">…</RequireAccess>          // butuh izin
//   <RequireAccess anyOf={['agen.view', 'komisi.view']}>…       // salah satu izin
//   <RequireAccess holdingOnly>…</RequireAccess>                // hanya akun holding
const RequireAccess = ({ perm, anyOf, holdingOnly = false, children }) => {
  const { hasPermission, isHolding } = useContext(AuthContext);

  let allowed = true;
  if (holdingOnly && !isHolding) allowed = false;
  if (perm && !hasPermission(perm)) allowed = false;
  if (anyOf && !anyOf.some((p) => hasPermission(p))) allowed = false;

  if (allowed) return children;

  return (
    <div className="mx-auto mt-16 max-w-md rounded-xl border border-neutral-200 bg-white p-8 text-center">
      <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-warning-50 text-warning-600">
        <ShieldAlert size={22} strokeWidth={1.75} />
      </div>
      <h2 className="font-heading text-base font-bold text-neutral-900">Anda tidak punya akses ke halaman ini</h2>
      <p className="mt-1.5 text-sm font-body text-neutral-500">
        Halaman ini tidak termasuk dalam hak akses akun Anda. Hubungi admin holding bila memerlukannya.
      </p>
      <Link
        to="/"
        className="mt-5 inline-flex items-center rounded-xl border border-neutral-200 bg-white px-4 py-2 text-sm font-semibold text-neutral-800 hover:bg-neutral-50"
      >
        Kembali ke Dashboard
      </Link>
    </div>
  );
};

export default RequireAccess;
