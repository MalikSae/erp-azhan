import api from './client';

export const listBookings = async (params = {}) => {
  const queryParams = new URLSearchParams();
  const jamaahId = params.jamaahId || params.jamaah_id;
  if (jamaahId) queryParams.append('jamaah_id', jamaahId);
  if (params.schedule_id) queryParams.append('schedule_id', params.schedule_id);
  if (params.status) queryParams.append('status', params.status);

  const qs = queryParams.toString();
  const url = qs ? `/api/admin/bookings?${qs}` : '/api/admin/bookings';
  const { data } = await api.get(url);
  return data;
};

export const getBooking = async (id) => {
  const { data } = await api.get(`/api/admin/bookings/${id}`);
  return data;
};

export const createBooking = async (payload) => {
  const { data } = await api.post('/api/admin/bookings', payload);
  return data;
};

export const updateBookingStatus = async (id, status) => {
  const { data } = await api.put(`/api/admin/bookings/${id}/status`, { status });
  return data;
};

export const listPayments = async (bookingId) => {
  const { data } = await api.get(`/api/admin/bookings/${bookingId}/payments`);
  return data;
};

export const createPayment = async (bookingId, payload) => {
  const { data } = await api.post(`/api/admin/bookings/${bookingId}/payments`, payload);
  return data;
};

export const updatePaymentStatus = async (paymentId, status) => {
  const { data } = await api.put(`/api/admin/payments/${paymentId}/status`, { status });
  return data;
};

export const deletePayment = async (paymentId) => {
  const { data } = await api.delete(`/api/admin/payments/${paymentId}`);
  return data;
};

export const addBookingAddon = async (bookingId, payload) => {
  const { data } = await api.post(`/api/admin/bookings/${bookingId}/addons`, payload);
  return data;
};

export const deleteBookingAddon = async (bookingId, addonId) => {
  const { data } = await api.delete(`/api/admin/bookings/${bookingId}/addons/${addonId}`);
  return data;
};

export const updateBookingDiskon = async (bookingId, payload) => {
  const { data } = await api.put(`/api/admin/bookings/${bookingId}/diskon`, payload);
  return data;
};

export const updateBookingProgress = async (bookingId, updates) => {
  const { data } = await api.put(`/api/admin/bookings/${bookingId}/progress`, updates);
  return data;
};

