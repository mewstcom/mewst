# typed: strict
# frozen_string_literal: true

class Repost < ApplicationRecord
  delegated_type :repostable, types: Repostable::TYPES, dependent: :destroy

  counter_culture :repostable, column_name: "reposts_count"

  has_one :post, as: :postable, touch: true

  validates :repostable_type, inclusion: {in: Repostable::TYPES}

  sig { returns(Integer) }
  def reposts_count
    0
  end
end
