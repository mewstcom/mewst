# typed: strict
# frozen_string_literal: true

class PhoneNumberForm < ApplicationForm
  attribute :phone_number_origin, :string
  attribute :phone_number, :string

  validates :phone_number, presence: true
end
