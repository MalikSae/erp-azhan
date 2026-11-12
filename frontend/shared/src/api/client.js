import axios from 'axios';

const client = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
});

function getToken() {
  return localStorage.getItem('erp_access_token') || localStorage.getItem('access_token');
}

function getRefreshToken() {
  return localStorage.getItem('erp_refresh_token') || localStorage.getItem('refresh_token');
}

function setToken(token) {
  if (localStorage.getItem('erp_access_token') !== null) {
    localStorage.setItem('erp_access_token', token);
  }
  if (localStorage.getItem('access_token') !== null || localStorage.getItem('erp_access_token') === null) {
    localStorage.setItem('access_token', token);
  }
}

function setRefreshToken(token) {
  if (localStorage.getItem('refresh_token') !== null && localStorage.getItem('erp_refresh_token') === null) {
    localStorage.setItem('refresh_token', token);
  } else {
    localStorage.setItem('erp_refresh_token', token);
  }
}

function removeTokens() {
  localStorage.removeItem('erp_access_token');
  localStorage.removeItem('erp_refresh_token');
  localStorage.removeItem('access_token');
  localStorage.removeItem('refresh_token');
}

client.interceptors.request.use((config) => {
  const token = getToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// refreshBerjalan: permintaan refresh token yang sedang berlangsung, bila ada.
let refreshBerjalan = null;

client.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    if (
      error.response?.status === 401 &&
      originalRequest.url !== '/api/auth/login' &&
      originalRequest.url !== '/api/auth/refresh' &&
      !originalRequest._retry
    ) {
      originalRequest._retry = true;

      const refreshToken = getRefreshToken();
      if (!refreshToken) {
        removeTokens();
        window.location.href = '/login';
        return Promise.reject(error);
      }

      try {
        // Beberapa permintaan yang ditolak 401 bersamaan memakai satu permintaan refresh yang sama.
        if (!refreshBerjalan) {
          refreshBerjalan = axios.post(`${import.meta.env.VITE_API_BASE_URL}/api/auth/refresh`, {
            refresh_token: refreshToken,
          }).finally(() => { refreshBerjalan = null; });
        }
        const res = await refreshBerjalan;

        const newAccessToken = res.data.access_token;
        setToken(newAccessToken);
        // Refresh token dirotasi tiap refresh: token lama tidak berlaku lagi.
        if (res.data.refresh_token) setRefreshToken(res.data.refresh_token);

        originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
        return client(originalRequest);
      } catch (refreshError) {
        removeTokens();
        window.location.href = '/login';
        return Promise.reject(refreshError);
      }
    }

    return Promise.reject(error);
  }
);

export default client;
