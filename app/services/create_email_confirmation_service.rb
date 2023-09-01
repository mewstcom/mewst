# typed: strict
# frozen_string_literal: true

class CreateEmailConfirmationService < ApplicationService
  class Input < T::Struct
    extend T::Sig

    const :email, String
    const :locale, String

    sig { params(form: Internal::EmailConfirmationForm).returns(Input) }
    def self.from_internal_form(form:)
      new(
        email: form.email.not_nil!,
        locale: form.locale.not_nil!
      )
    end
  end

  class Result < T::Struct
    const :email_confirmation, EmailConfirmation
  end

  sig { params(input: Input).returns(Result) }
  def call(input:)
    email_confirmation = EmailConfirmation.create!(
      email: input.email,
      code: EmailConfirmation.generate_code,
      event: EmailConfirmation::EVENT_SIGN_UP
    )

    EmailConfirmationMailer.email_confirmation(
      email_confirmation_id: email_confirmation.id,
      locale: input.locale
    ).deliver_later

    Result.new(email_confirmation:)
  end
end
