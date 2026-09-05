import { createContext, useContext, useState, useEffect } from 'react';
import { jwtDecode } from 'jwt-decode';
import { getMyBrand } from 'shared';

const AuthContext = createContext();

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [brandInfo, setBrandInfo] = useState(null);
  const [loading, setLoading] = useState(true);

  const fetchBrandData = async () => {
    try {
      const res = await getMyBrand();
      setBrandInfo(res);
    } catch (e) {
      setBrandInfo(null);
    }
  };

  useEffect(() => {
    const token = localStorage.getItem('access_token') || localStorage.getItem('erp_access_token');
    if (token) {
      try {
        const decoded = jwtDecode(token);
        setUser({
          id: decoded.sub ?? decoded.user_id,
          brand_id: decoded.brand_id,
          role: decoded.role,
          email: decoded.email,
        });
        fetchBrandData();
      } catch (e) {
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
      }
    }
    setLoading(false);
  }, []);

  const login = async (accessToken, refreshToken) => {
    try {
      const decoded = jwtDecode(accessToken);

      localStorage.setItem('access_token', accessToken);
      localStorage.setItem('refresh_token', refreshToken);
      setUser({
        id: decoded.sub ?? decoded.user_id,
        brand_id: decoded.brand_id,
        role: decoded.role,
        email: decoded.email,
      });
      
      await fetchBrandData();
      
      return { success: true };
    } catch (e) {
      return { success: false, message: "Token tidak valid" };
    }
  };

  const logout = () => {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('erp_access_token');
    localStorage.removeItem('erp_refresh_token');
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
