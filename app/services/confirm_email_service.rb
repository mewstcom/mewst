# typed: strict
# frozen_string_literal: true

class ConfirmEmailService < ApplicationService
  class Input < T::Struct
    extend T::Sig

    const :email_confirmation, EmailConfirmation

    sig { params(form: Internal::EmailConfirmationChallengeForm).returns(Input) }
    def self.from_internal_form(form:)
      new(email_confirmation: form.email_confirmation!)
    end
  end

  sig { params(input: Input).void }
  def call(input:)
    input.email_confirmation.success
  end
end
