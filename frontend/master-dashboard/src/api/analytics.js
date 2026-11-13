import client from './client';

export async function listTransactions30Days() {
  const response = await client.get('/api/admin/analytics/transactions-30-days');
  return response.data;
}

export async function listPax30Days() {
  const response = await client.get('/api/admin/analytics/pax-30-days');
  return response.data;
}

// Ringkasan KPI marketing Dashboard Utama: pax per channel, seat terisi 60 hari, 10 agen terbaik.
export async function getMarketingRingkas() {
  const response = await client.get('/api/admin/analytics/marketing-ringkas');
  return response.data;
}
