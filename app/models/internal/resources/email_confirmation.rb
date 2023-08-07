# typed: strict
# frozen_string_literal: true

class Internal::Resources::EmailConfirmation < Internal::Resources::Base
  root_key :email_confirmation, :email_confirmations

  attributes :email, :id, :succeeded_at
end
