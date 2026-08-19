import axios from 'axios';

const client = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
});

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('erp_access_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

client.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;
    
    // Jika 401 dan bukan saat login/refresh
    if (
      error.response?.status === 401 &&
      originalRequest.url !== '/api/auth/login' &&
      originalRequest.url !== '/api/auth/refresh' &&
      !originalRequest._retry
    ) {
      originalRequest._retry = true;
      
      const refreshToken = localStorage.getItem('erp_refresh_token');
      if (!refreshToken) {
        // Tidak ada refresh token, clear & redirect
        localStorage.removeItem('erp_access_token');
        localStorage.removeItem('erp_refresh_token');
        window.location.href = '/login';
        return Promise.reject(error);
      }
      
      try {
        const res = await axios.post(`${import.meta.env.VITE_API_BASE_URL}/api/auth/refresh`, {
          refresh_token: refreshToken,
        });
        
        const newAccessToken = res.data.access_token;
        localStorage.setItem('erp_access_token', newAccessToken);
        
        // Update header & retry
        originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
        return client(originalRequest);
        
      } catch (refreshError) {
        // Refresh gagal (misal 401), clear & redirect
        localStorage.removeItem('erp_access_token');
        localStorage.removeItem('erp_refresh_token');
        window.location.href = '/login';
        return Promise.reject(refreshError);
      }
    }
    
    return Promise.reject(error);
  }
);

export default client;
