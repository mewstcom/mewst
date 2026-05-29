# typed: strict
# frozen_string_literal: true

class V1::OauthAccessTokenSerializer < V1::ApplicationSerializer
  root_key :oauth_access_token, :oauth_access_tokens

  attributes :token
end
