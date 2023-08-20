# typed: strict
# frozen_string_literal: true

class Stamp < ApplicationRecord
  counter_culture :post

  belongs_to :post
  belongs_to :profile
end
