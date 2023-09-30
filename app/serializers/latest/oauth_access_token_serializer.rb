# typed: strict
# frozen_string_literal: true

class Latest::OauthAccessTokenSerializer < Latest::ApplicationSerializer
  root_key :oauth_access_token, :oauth_access_tokens

  attributes :token
end
