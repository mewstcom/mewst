# typed: strict
# frozen_string_literal: true

class Commands::SendEmailConfirmationCode < Commands::Base
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
    SendEmailConfirmationMailJob.perform_later(email_confirmation_id: email_confirmation.id, locale: form.locale)

    Result.new(email_confirmation:)
  end

  private

  sig { returns(Forms::EmailConfirmation) }
  attr_reader :form
end
