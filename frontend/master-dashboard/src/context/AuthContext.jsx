import React, { createContext, useState, useEffect } from 'react';
import client from '../api/client';

export const AuthContext = createContext();

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
  
  const [currentBrandId, setCurrentBrandId] = useState(null);
  const [currentUserId, setCurrentUserId] = useState(null);

  useEffect(() => {
    if (accessToken) {
      const payload = decodeJwtPayload(accessToken);
      if (payload) {
        setCurrentBrandId(payload.brand_id);
        setCurrentUserId(payload.sub ? Number(payload.sub) : null);
      } else {
        setCurrentBrandId(null);
        setCurrentUserId(null);
      }
    } else {
      setCurrentBrandId(null);
      setCurrentUserId(null);
    }
  }, [accessToken]);

  const login = async (email, password) => {
    try {
      const res = await client.post('/api/auth/login', { email, password });
      const { access_token, refresh_token } = res.data;
      
      const payload = decodeJwtPayload(access_token);
      if (payload && payload.brand_id !== null) {
        throw new Error("Master Dashboard khusus untuk Super Admin Grup. Akun ini terikat ke brand tertentu — gunakan Travel Dashboard (belum tersedia di fase ini).");
      }

      localStorage.setItem('erp_access_token', access_token);
      localStorage.setItem('erp_refresh_token', refresh_token);
      setAccessToken(access_token);
      
      return true;
    } catch (error) {
      if (error.message === "Master Dashboard khusus untuk Super Admin Grup. Akun ini terikat ke brand tertentu — gunakan Travel Dashboard (belum tersedia di fase ini).") {
        throw error;
      }
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
    <AuthContext.Provider value={{ accessToken, login, logout, currentBrandId, currentUserId }}>
      {children}
    </AuthContext.Provider>
  );
};
