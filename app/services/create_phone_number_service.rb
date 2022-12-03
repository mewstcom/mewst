# typed: strict
# frozen_string_literal: true

class CreatePhoneNumberService < ApplicationService
  class Error < T::Struct
    const :message, String
  end

  class Result < T::Struct
    const :errors, T::Array[Error], default: []
    const :phone_number, T.nilable(PhoneNumber)
  end

  sig { params(form: SmsCodeForm).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    phone_number = ActiveRecord::Base.transaction do
      phone_number = PhoneNumber.find_or_create_by!(value: @form.phone_number_confirmation.phone_number)
      @form.phone_number_confirmation.destroy

      phone_number
    end

    Result.new(phone_number:)
  end
end
