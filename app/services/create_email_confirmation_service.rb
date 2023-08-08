# typed: strict
# frozen_string_literal: true

class CreateEmailConfirmationService < ApplicationService
  class Result < T::Struct
    const :email_confirmation, EmailConfirmation
  end

  sig { params(form: Forms::EmailConfirmation).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    email_confirmation = EmailConfirmation.create!(
      email: form.email,
      code: EmailConfirmation.generate_code,
      event: EmailConfirmation::EVENT_SIGN_UP
    )
    SendEmailConfirmationMailJob.perform_later(email_confirmation_id: email_confirmation.id, locale: T.must(form.locale))

    Result.new(email_confirmation:)
  end

  sig { returns(Forms::EmailConfirmation) }
  attr_reader :form
  private :form
end
