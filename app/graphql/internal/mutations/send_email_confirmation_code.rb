# typed: strict
# frozen_string_literal: true

class Internal::Mutations::SendEmailConfirmationCode < Internal::Mutations::Base
  argument :email, String, required: true

  field :email_confirmation_id, String, null: true
  field :errors, [Internal::Types::Objects::ClientErrorType], null: false

  def resolve(email:)
    form = Forms::EmailConfirmation.new(email:, locale: context[:current_locale])

    if form.invalid?
      return {
        email_confirmation_id: nil,
        errors: form.errors.full_messages.map { |message| {message:} }
      }
    end

    result = Commands::SendEmailConfirmationCode.new(form:).call

    {
      email_confirmation_id: result.email_confirmation.id,
      errors: []
    }
  end
end
