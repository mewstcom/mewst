# typed: strict
# frozen_string_literal: true

class Internal::Types::Objects::OauthAccessToken < Internal::Types::Objects::Base
  field :database_id, String, null: false
  field :token, String, null: false

  def database_id
    object.id
  end
end
