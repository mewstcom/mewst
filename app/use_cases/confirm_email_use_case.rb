# typed: strict
# frozen_string_literal: true

class ConfirmEmailUseCase < ApplicationUseCase
  sig { params(email_confirmation: EmailConfirmation).void }
  def call(email_confirmation:)
    email_confirmation.success
  end
end
