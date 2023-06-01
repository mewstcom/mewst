# typed: strict
# frozen_string_literal: true

class Internal::Types::Objects::QueryType < Internal::Types::Objects::Base
  field :succeeded_email_confirmation, Internal::Types::Objects::EmailConfirmationType, null: true do
    argument :email_confirmation_id, String, required: true
  end

  def succeeded_email_confirmation(email_confirmation_id:)
    EmailConfirmation.succeeded.find_by(id: email_confirmation_id)
  end
end
