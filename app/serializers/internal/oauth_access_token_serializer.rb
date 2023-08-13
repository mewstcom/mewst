# typed: strict
# frozen_string_literal: true

class Internal::OauthAccessTokenSerializer < Internal::ApplicationSerializer
  root_key :oauth_access_token, :oauth_access_tokens

  attributes :token
end
