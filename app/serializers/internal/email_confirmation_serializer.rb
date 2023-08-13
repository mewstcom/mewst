# typed: strict
# frozen_string_literal: true

class Internal::EmailConfirmationSerializer < Internal::ApplicationSerializer
  root_key :email_confirmation, :email_confirmations

  attributes :email, :id, :succeeded_at
end
