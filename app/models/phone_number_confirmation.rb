# typed: strict
# frozen_string_literal: true

class PhoneNumberConfirmation < ApplicationRecord
  validates :phone_number, presence: true
  validates :sms_code, presence: true

  sig { returns(String) }
  def self.generate_sms_code
    6.times.map { rand(10) }.join
  end
end
