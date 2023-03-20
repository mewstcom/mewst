# typed: strict
# frozen_string_literal: true

class VerificationMailer < ApplicationMailer
  sig { params(verification_id: String, locale: String).void }
  def email_verification(verification_id:, locale:)
    verification = Verification.active.find(verification_id)

    @email = verification.email
    @code = verification.code

    I18n.with_locale(locale) do
      subject = default_i18n_subject
      mail(to: @email, subject:)
    end
  end
end
