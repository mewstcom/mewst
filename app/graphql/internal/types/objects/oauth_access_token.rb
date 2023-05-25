# typed: strict
# frozen_string_literal: true

class Internal::Types::Objects::OauthAccessToken < Internal::Types::Objects::Base
  field :token, String, null: false
end
