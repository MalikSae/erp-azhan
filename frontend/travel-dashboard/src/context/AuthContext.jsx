import { createContext, useContext, useState, useEffect } from 'react';
import { jwtDecode } from 'jwt-decode';
import { getMyBrand } from 'shared';

const AuthContext = createContext();
const USER_EMAIL_STORAGE_KEY = 'travel_user_email';

const userFromToken = (token, fallbackEmail = '') => {
  const decoded = jwtDecode(token);
  return {
    id: decoded.sub ?? decoded.user_id,
    brand_id: decoded.brand_id,
    role: decoded.role,
    email: decoded.email || fallbackEmail || localStorage.getItem(USER_EMAIL_STORAGE_KEY) || '',
  };
};

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [brandInfo, setBrandInfo] = useState(null);
  const [loading, setLoading] = useState(true);

  const fetchBrandData = async () => {
    try {
      const res = await getMyBrand();
      setBrandInfo(res);
    } catch {
      setBrandInfo(null);
    }
  };

  useEffect(() => {
    const token = localStorage.getItem('access_token') || localStorage.getItem('erp_access_token');
    if (token) {
      try {
        setUser(userFromToken(token));
        fetchBrandData();
      } catch {
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
      }
    }
    setLoading(false);
  }, []);

  const login = async (accessToken, refreshToken, profileEmail = '') => {
    try {
      localStorage.setItem('access_token', accessToken);
      localStorage.setItem('refresh_token', refreshToken);
      if (profileEmail) localStorage.setItem(USER_EMAIL_STORAGE_KEY, profileEmail);
      setUser(userFromToken(accessToken, profileEmail));
      
      await fetchBrandData();
      
      return { success: true };
    } catch {
      return { success: false, message: "Token tidak valid" };
    }
  };

  const logout = () => {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('erp_access_token');
    localStorage.removeItem('erp_refresh_token');
    localStorage.removeItem(USER_EMAIL_STORAGE_KEY);
    setUser(null);
    setBrandInfo(null);
  };

  if (loading) return null;

  return (
    <AuthContext.Provider value={{ user, brandInfo, login, logout, refreshBrand: fetchBrandData }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);
