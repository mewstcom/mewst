# typed: strict
# frozen_string_literal: true

class ConfirmEmailUseCase < ApplicationUseCase
  class Result < T::Struct
    const :email_confirmation, EmailConfirmation
  end

  sig { params(current_actor: T.nilable(Actor), email_confirmation: EmailConfirmation).returns(Result) }
  def call(current_actor:, email_confirmation:)
    ActiveRecord::Base.transaction do
      email_confirmation.success!
      email_confirmation.process_after_success!(current_actor:)
    end

    Result.new(email_confirmation:)
  end
end
