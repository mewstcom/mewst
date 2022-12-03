# typed: strict
# frozen_string_literal: true

class SmsCodeForm < ApplicationForm
  attribute :sms_code, :string

  attr_accessor :phone_number_confirmation

  validates :sms_code, presence: true
  validate :sms_code_equal_to_phone_number_confirmation_sms_code

  private def sms_code_equal_to_phone_number_confirmation_sms_code
    if sms_code != phone_number_confirmation&.sms_code
      errors.add(:sms_code, :mismatch)
    end
  end
end
