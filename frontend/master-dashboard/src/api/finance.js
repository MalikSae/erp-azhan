import client from './client';

export async function getFinanceSummary() {
  const response = await client.get('/api/admin/finance/ringkasan');
  return response.data;
}
