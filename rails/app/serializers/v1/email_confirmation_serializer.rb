# typed: strict
# frozen_string_literal: true

class V1::EmailConfirmationSerializer < V1::ApplicationSerializer
  root_key :email_confirmation, :email_confirmations

  attributes :id, :email, :event, :succeeded_at
end
