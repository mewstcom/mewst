# typed: strict
# frozen_string_literal: true

class CreateEmailConfirmationUseCase < ApplicationUseCase
  class Result < T::Struct
    const :email_confirmation, EmailConfirmation
  end

  sig { params(email: String, event: String, locale: String).returns(Result) }
  def call(email:, event:, locale:)
    email_confirmation = EmailConfirmation.new(email:, event:, code: EmailConfirmation.generate_code)

    ActiveRecord::Base.transaction do
      email_confirmation.save!

      EmailConfirmationMailer.email_confirmation(
        email_confirmation_id: email_confirmation.id.not_nil!,
        locale:
      ).deliver_later
    end

    Result.new(email_confirmation:)
  end
end
