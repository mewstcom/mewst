# typed: false
# frozen_string_literal: true

require "shrine"

if ENV.fetch("MEWST_CLOUDFLARE_R2_ACCESS_KEY_ID", false)
  require "shrine/storage/s3"

  s3_options = {
    bucket: ENV.fetch("MEWST_CLOUDFLARE_R2_BUCKET_NAME"),
    access_key_id: ENV.fetch("MEWST_CLOUDFLARE_R2_ACCESS_KEY_ID"),
    secret_access_key: ENV.fetch("MEWST_CLOUDFLARE_R2_SECRET_ACCESS_KEY"),
    region: ENV.fetch("MEWST_CLOUDFLARE_R2_REGION"),
    endpoint: ENV.fetch("MEWST_CLOUDFLARE_R2_ENDPOINT")
  }

  Shrine.storages = {
    cache: Shrine::Storage::S3.new(prefix: "cache", **s3_options),
    store: Shrine::Storage::S3.new(**s3_options),
  }
else
  require "shrine/storage/file_system"

  Shrine.storages = {
    cache: Shrine::Storage::FileSystem.new("public", prefix: "uploads/cache"),
    store: Shrine::Storage::FileSystem.new("public", prefix: "uploads")
  }
end

Shrine.logger = Rails.logger

Shrine.plugin(:activerecord)
Shrine.plugin(:cached_attachment_data)
Shrine.plugin(:restore_cached_data)
Shrine.plugin(:derivation_endpoint, secret_key: ENV.fetch("MEWST_SHRINE_SECRET_KEY"))
