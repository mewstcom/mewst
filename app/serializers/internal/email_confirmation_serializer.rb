# typed: strict
# frozen_string_literal: true

class Internal::EmailConfirmationSerializer < Internal::ApplicationSerializer
  attributes :email, :id, :succeeded_at
end
