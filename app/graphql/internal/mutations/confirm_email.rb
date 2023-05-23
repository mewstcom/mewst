# typed: strict
# frozen_string_literal: true

class Internal::Mutations::ConfirmEmail < Internal::Mutations::Base
  argument :email_confirmation_id, String, required: true
  argument :confirmation_code, String, required: true

  field :errors, [Internal::Types::Objects::ClientErrorType], null: false

  def resolve(email_confirmation_id:, confirmation_code:)
    form = Forms::EmailConfirmationChallenge.new(email_confirmation_id:, confirmation_code:)

    if form.invalid?
      return {
        errors: form.errors.full_messages.map { |message| {message:} }
      }
    end

    Commands::ConfirmEmail.new(form:).call

    {
      errors: []
    }
  end
end
