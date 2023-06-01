# typed: strict
# frozen_string_literal: true

class Internal::Types::Objects::OauthAccessTokenType < Internal::Types::Objects::Base
  implements GraphQL::Types::Relay::Node

  field :token, String, null: false
end
