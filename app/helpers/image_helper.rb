# typed: strict
# frozen_string_literal: true

module ImageHelper
  extend T::Sig

  sig { params(record: ApplicationRecord, field: Symbol, width: Integer).returns(String) }
  def mst_image_url(record, field, width:)
    cdn_image_url(record.send(field).variant(resize_to_limit: [width, nil], format: :webp, saver: {quality: 80}))
  end
end
