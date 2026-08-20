ALTER TABLE schedules
  ADD COLUMN berangkat_bandara_asal VARCHAR(150) NULL AFTER berangkat_kode_penerbangan,
  ADD COLUMN berangkat_bandara_tujuan VARCHAR(150) NULL AFTER berangkat_bandara_asal,
  ADD COLUMN pulang_bandara_asal VARCHAR(150) NULL AFTER pulang_kode_penerbangan,
  ADD COLUMN pulang_bandara_tujuan VARCHAR(150) NULL AFTER pulang_bandara_asal,
  ADD COLUMN transit_bandara VARCHAR(255) NULL AFTER pulang_bandara_tujuan;
