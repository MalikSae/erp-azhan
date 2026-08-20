import client from './client';
export const listBankAccounts = async () => (await client.get('/api/admin/bank-accounts/')).data;
export const createBankAccount = async (payload) => (await client.post('/api/admin/bank-accounts/', payload)).data;
export const updateBankAccount = async (id,payload) => (await client.put(`/api/admin/bank-accounts/${id}`,payload)).data;
export const deleteBankAccount = async (id) => (await client.delete(`/api/admin/bank-accounts/${id}`)).data;
