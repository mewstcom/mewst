# typed: strict
# frozen_string_literal: true

class ConfirmEmailUseCase < ApplicationUseCase
  class Result < T::Struct
    const :email_confirmation, EmailConfirmationRecord
  end

  sig { params(viewer: T.nilable(ActorRecord), email_confirmation: EmailConfirmationRecord).returns(Result) }
  def call(viewer:, email_confirmation:)
    ActiveRecord::Base.transaction do
      email_confirmation.success!
      email_confirmation.process_after_success!(viewer:)
    end

    Result.new(email_confirmation:)
  end
end
