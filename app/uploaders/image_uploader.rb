# typed: strict
# frozen_string_literal: true

class ImageUploader < Shrine
  ALLOWED_TYPES = T.let(%w[image/gif image/jpeg image/png].freeze, T::Array[String])
  MAX_SIZE = T.let(10.megabytes, Integer)
  MAX_SIDE_LENGTH = T.let(5_000, Integer)

  plugin :remove_attachment
  plugin :pretty_location
  plugin :processing
  plugin :versions
  plugin :validation_helpers
  plugin :store_dimensions, analyzer: :mini_magick
  plugin :derivation_endpoint, prefix: "image", secret_key: ENV.fetch("MEWST_SHRINE_SECRET_KEY"), type: "image/webp"

  Attacher.validate do
    validate_max_size MAX_SIZE

    if validate_mime_type_inclusion(ALLOWED_TYPES)
      validate_max_width MAX_SIDE_LENGTH
      validate_max_height MAX_SIDE_LENGTH
    end
  end

  process(:store) do |io, _context|
    unless io.original_filename
      ext = MIME::Types[io.metadata["mime_type"]].first.extensions.first
      io.data["metadata"]["filename"] = "#{io.hash}.#{ext}" if ext
    end

    versions = {original: io}

    io.download do |original|
      versions[:master] = ImageProcessing::MiniMagick
        .source(original)
        .loader(page: 0) # For gif animation image
        .convert(:jpg)
        .saver(quality: 90)
        .strip
        .resize_to_limit(1000, nil, sharpen: false)
        .call
    end

    versions
  end

  derivation :thumbnail do |file, width|
    ImageProcessing::MiniMagick
      .source(file)
      .resize_to_limit!(width.to_i, nil)
  end
end
