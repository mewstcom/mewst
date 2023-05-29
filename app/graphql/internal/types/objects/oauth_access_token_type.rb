# typed: strict
# frozen_string_literal: true

class Internal::Types::Objects::OauthAccessTokenType < Internal::Types::Objects::Base
  field :database_id, String, null: false
  field :token, String, null: false
  field :account, Internal::Types::Objects::AccountType, null: false
  field :profile, Internal::Types::Objects::ProfileType, null: false

  def database_id
    object.id
  end
end
