import client from './client';

export async function listTransactions30Days() {
  const response = await client.get('/api/admin/analytics/transactions-30-days');
  return response.data;
}

export async function listPax30Days() {
  const response = await client.get('/api/admin/analytics/pax-30-days');
  return response.data;
}
