# typed: strict
# frozen_string_literal: true

module ImageHelper
  extend T::Sig

  sig { params(record: ApplicationRecord, field: Symbol, width: Integer).returns(String) }
  def mst_image_url(record, field, width:)
    record.send(field).derivation_url(:thumbnail, width)
  end
end
