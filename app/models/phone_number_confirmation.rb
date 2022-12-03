# typed: strict
# frozen_string_literal: true

class PhoneNumberConfirmation < ApplicationRecord
  validates :code, presence: true
  validates :phone_number, presence: true
  validates :phone_number_full, presence: true

  sig { returns(String) }
  def self.generate_code
    6.times.map { rand(10) }.join
  end
end
