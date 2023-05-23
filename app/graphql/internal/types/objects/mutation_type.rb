# typed: strict
# frozen_string_literal: true

class Internal::Types::Objects::MutationType < Internal::Types::Objects::Base
  field :confirm_email, mutation: Internal::Mutations::ConfirmEmail
  field :send_email_confirmation_code, mutation: Internal::Mutations::SendEmailConfirmationCode
end
