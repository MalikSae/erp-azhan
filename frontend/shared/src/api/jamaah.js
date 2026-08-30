import api from './client';

export const listJamaah = async (params = {}) => {
  const { data } = await api.get('/api/admin/jamaah', { params });
  return data;
};

export const getJamaah = async (id) => {
  const { data } = await api.get(`/api/admin/jamaah/${id}`);
  return data;
};

export const createJamaah = async (payload) => {
  const { data } = await api.post('/api/admin/jamaah', payload);
  return data;
};

export const updateJamaah = async (id, payload) => {
  const { data } = await api.put(`/api/admin/jamaah/${id}`, payload);
  return data;
};

export const deleteJamaah = async (id) => {
  const { data } = await api.delete(`/api/admin/jamaah/${id}`);
  return data;
};

export const updateCatatan = async (id, catatan) => {
  const { data } = await api.put(`/api/admin/jamaah/${id}/catatan`, { catatan });
  return data;
};

export const listRelasi = async (jamaahId) => {
  const { data } = await api.get(`/api/admin/jamaah/${jamaahId}/relasi`);
  return data;
};

export const createRelasi = async (jamaahId, payload) => {
  const { data } = await api.post(`/api/admin/jamaah/${jamaahId}/relasi`, payload);
  return data;
};

export const deleteRelasi = async (jamaahId, relasiId) => {
  const { data } = await api.delete(`/api/admin/jamaah/${jamaahId}/relasi/${relasiId}`);
  return data;
};

