import api from './client';

export const listJamaah = async () => {
  const { data } = await api.get('/api/admin/jamaah');
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
