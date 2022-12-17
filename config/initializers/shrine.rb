# typed: false
# frozen_string_literal: true

require "shrine"
require "shrine/storage/file_system"

Shrine.storages = {
  cache: Shrine::Storage::FileSystem.new("public", prefix: "uploads/cache"),
  store: Shrine::Storage::FileSystem.new("public", prefix: "uploads")
}

Shrine.logger = Rails.logger

Shrine.plugin(:activerecord)
Shrine.plugin(:cached_attachment_data)
Shrine.plugin(:restore_cached_data)
Shrine.plugin(:derivation_endpoint, secret_key: ENV.fetch("MEWST_SHRINE_SECRET_KEY"))
