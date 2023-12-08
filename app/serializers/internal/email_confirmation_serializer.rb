# typed: strict
# frozen_string_literal: true

class Internal::EmailConfirmationSerializer < Internal::ApplicationSerializer
  root_key :email_confirmation, :email_confirmations

  attributes :id, :email, :event, :succeeded_at
end
