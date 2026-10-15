import React, { createContext, useState, useEffect, useMemo, useContext, useCallback } from 'react';
import client from '../api/client';
import { getMyBrand } from 'shared';

export const AuthContext = createContext();

// Nama peran untuk tampilan (slug dari JWT -> label).
export const ROLE_LABEL = {
  super_admin_grup: 'Super Admin',
  direktur: 'Direktur / Owner',
  cmo: 'CMO',
  marketing_lead: 'Marketing Lead',
  marketing_member: 'Marketing Member',
  product_manager: 'Product Manager',
  admin_travel: 'Admin Brand',
  admin_cabang: 'Admin Cabang',
  cs: 'Customer Service',
  finance: 'Finance',
};

function decodeJwtPayload(token) {
  try {
    const base64Url = token.split('.')[1];
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
    const jsonPayload = decodeURIComponent(atob(base64).split('').map(function(c) {
      return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
    }).join(''));
    return JSON.parse(jsonPayload);
  } catch (error) {
    return null;
  }
}

export const AuthProvider = ({ children }) => {
  const [accessToken, setAccessToken] = useState(
    localStorage.getItem('erp_access_token') || null
  );
  const [brandInfo, setBrandInfo] = useState(null);

  const payload = useMemo(() => (accessToken ? decodeJwtPayload(accessToken) : null), [accessToken]);

  const currentBrandId = payload ? (payload.brand_id ?? null) : null;
  const currentUserId = payload?.sub ? Number(payload.sub) : null;
  const permissions = useMemo(() => (Array.isArray(payload?.perms) ? payload.perms : []), [payload]);
  const roles = useMemo(() => (Array.isArray(payload?.roles) ? payload.roles : []), [payload]);
  // Holding = akun tanpa ikatan brand; admin brand hanya melihat data brand-nya.
  const isHolding = currentBrandId === null;

  const hasPermission = useCallback((code) => permissions.includes(code), [permissions]);

  // Admin brand: ambil identitas brand-nya (nama, logo) untuk header & halaman.
  useEffect(() => {
    if (!accessToken || isHolding) {
      setBrandInfo(null);
      return;
    }
    let cancelled = false;
    getMyBrand()
      .then((b) => { if (!cancelled) setBrandInfo(b); })
      .catch(() => { if (!cancelled) setBrandInfo(null); });
    return () => { cancelled = true; };
  }, [accessToken, isHolding]);

  const user = useMemo(() => {
    if (!payload) return null;
    return {
      id: currentUserId,
      email: payload.email || '',
      name: payload.display_name || (payload.email ? payload.email.split('@')[0] : ''),
      role: roles[0] || payload.role || '',
      roleLabel: ROLE_LABEL[roles[0]] || (isHolding ? 'Holding' : 'Admin Brand'),
    };
  }, [payload, currentUserId, roles, isHolding]);

  const login = async (email, password) => {
    try {
      const res = await client.post('/api/auth/login', { email, password });
      const { access_token, refresh_token } = res.data;

      localStorage.setItem('erp_access_token', access_token);
      localStorage.setItem('erp_refresh_token', refresh_token);
      setAccessToken(access_token);

      return true;
    } catch (error) {
      if (error.response && error.response.data && error.response.data.error) {
        throw new Error(error.response.data.error);
      }
      throw new Error('Gagal login. Coba lagi.');
    }
  };

  const logout = async () => {
    try {
      await client.post('/api/auth/logout', {
        refresh_token: localStorage.getItem('erp_refresh_token')
      });
    } catch (error) {
      // fire and forget
    } finally {
      localStorage.removeItem('erp_access_token');
      localStorage.removeItem('erp_refresh_token');
      setAccessToken(null);
      window.location.href = '/login';
    }
  };

  return (
    <AuthContext.Provider
      value={{
        accessToken,
        login,
        logout,
        currentBrandId,
        currentUserId,
        isHolding,
        permissions,
        hasPermission,
        brandInfo,
        user,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);
