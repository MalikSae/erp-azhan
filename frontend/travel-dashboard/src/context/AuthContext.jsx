import { createContext, useContext, useState, useEffect } from 'react';
import { jwtDecode } from 'jwt-decode';
import { getMyBrand } from '../api/brand';

const AuthContext = createContext();

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [brandInfo, setBrandInfo] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const token = localStorage.getItem('access_token');
    if (token) {
      try {
        const decoded = jwtDecode(token);
        if (decoded.brand_id !== null) {
          setUser({
            id: decoded.user_id,
            brand_id: decoded.brand_id,
            role: decoded.role,
            email: decoded.email,
          });
          getMyBrand().then(res => setBrandInfo(res)).catch(() => setBrandInfo(null));
        } else {
          localStorage.removeItem('access_token');
          localStorage.removeItem('refresh_token');
        }
      } catch (e) {
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
      }
    }
    setLoading(false);
  }, []);

  const login = (accessToken, refreshToken) => {
    try {
      const decoded = jwtDecode(accessToken);
      // REVERSED GUARD: Reject if brand_id is null (Super Admin)
      if (decoded.brand_id === null) {
        return {
          success: false,
          message: "Travel Dashboard khusus untuk Admin Travel per-brand. Gunakan Master Dashboard untuk akses Super Admin Grup."
        };
      }

      localStorage.setItem('access_token', accessToken);
      localStorage.setItem('refresh_token', refreshToken);
      setUser({
        id: decoded.user_id,
        brand_id: decoded.brand_id,
        role: decoded.role,
        email: decoded.email,
      });
      
      // Async fetch but don't block the login return
      getMyBrand().then(res => setBrandInfo(res)).catch(() => setBrandInfo(null));
      
      return { success: true };
    } catch (e) {
      return { success: false, message: "Token tidak valid" };
    }
  };

  const logout = () => {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    setUser(null);
    setBrandInfo(null);
  };

  if (loading) return null;

  return (
    <AuthContext.Provider value={{ user, brandInfo, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);
