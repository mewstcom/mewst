# typed: strict
# frozen_string_literal: true

class ConfirmEmailUseCase < ApplicationUseCase
  class Result < T::Struct
    const :email_confirmation, EmailConfirmation
  end

  sig { params(email_confirmation: EmailConfirmation).returns(Result) }
  def call(email_confirmation:)
    email_confirmation.success!

    Result.new(email_confirmation:)
  end
end
