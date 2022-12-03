# typed: strict
# frozen_string_literal: true

class PhoneNumber < ApplicationRecord
  has_one :user_phone_number, dependent: :restrict_with_exception
  has_one :user, dependent: :restrict_with_exception, through: :user_phone_number

  validates :value, presence: true
end
