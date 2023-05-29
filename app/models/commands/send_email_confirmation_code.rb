# typed: strict
# frozen_string_literal: true

class Commands::SendEmailConfirmationCode < Commands::Base
  class Result < T::Struct
    const :email_confirmation, Verification
  end

  sig { params(form: Forms::EmailConfirmation).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    email_confirmation = Verification.create!(
      email: form.email,
      code: Verification.generate_code,
      event: Verification::EVENT_SIGN_UP
    )
    SendVerificationMailJob.perform_later(verification_id: email_confirmation.id, locale: form.locale)

    Result.new(email_confirmation:)
  end

  private

  sig { returns(Forms::EmailConfirmation) }
  attr_reader :form
end
