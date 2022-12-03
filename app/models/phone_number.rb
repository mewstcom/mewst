# typed: strict
# frozen_string_literal: true

class PhoneNumber < ApplicationRecord
  validates :value, presence: true
end
