# typed: strict
# frozen_string_literal: true

class Resources::Internal::EmailConfirmation < Resources::Internal::Base
  root_key :email_confirmation, :email_confirmations

  attributes :email, :id, :succeeded_at
end
