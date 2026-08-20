import client from './client';
export const listPayments = async (status='') => (await client.get('/api/admin/payments',{params:status?{status}:{}})).data;
export const verifyPayment = async (id,status,rejectionReason=null) => (await client.put(`/api/admin/payments/${id}/status`,{status,rejection_reason:rejectionReason})).data;
