import client from './client';

export const getBrandSurat = async (brandId) => (await client.get(`/api/admin/brands/${brandId}/surat`)).data;
export const updateBrandSurat = async (brandId, payload) => (await client.put(`/api/admin/brands/${brandId}/surat`, payload)).data;
