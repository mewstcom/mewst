# typed: strict
# frozen_string_literal: true

class CreateEmailConfirmationUseCase < ApplicationUseCase
  class Result < T::Struct
    const :email_confirmation, EmailConfirmation
  end

  sig { params(email: String, locale: String).returns(Result) }
  def call(email:, locale:)
    email_confirmation = EmailConfirmation.new(email:)

    ActiveRecord::Base.transaction do
      email_confirmation.send_sign_up_email!(locale:)
    end

    Result.new(email_confirmation:)
  end
end
