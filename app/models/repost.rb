# typed: strict
# frozen_string_literal: true

class Repost < ApplicationRecord
  counter_culture :repostable, column_name: "reposts_count"

  has_one :post, as: :postable, dependent: :restrict_with_exception, touch: true

  sig { returns(Integer) }
  def reposts_count
    0
  end
end
