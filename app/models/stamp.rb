# typed: strict
# frozen_string_literal: true

class Stamp < ApplicationRecord
  delegated_type :stampable, types: Stampable::TYPES, dependent: :destroy

  counter_culture :stampable, column_name: "stamps_count"

  belongs_to :post
  belongs_to :profile

  validates :stampable_type, inclusion: {in: Stampable::TYPES}
end
