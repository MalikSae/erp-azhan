import api from "./client";

export const getMyBrand = async () => {
  const response = await api.get("/api/admin/my-brand");
  return response.data;
};
