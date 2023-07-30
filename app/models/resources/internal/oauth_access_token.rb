# typed: strict
# frozen_string_literal: true

class Resources::Internal::OauthAccessToken < Resources::Internal::Base
  root_key :oauth_access_token, :oauth_access_tokens

  attributes :token
end
