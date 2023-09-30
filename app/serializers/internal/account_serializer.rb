# typed: strict
# frozen_string_literal: true

class Internal::AccountSerializer < Internal::ApplicationSerializer
  root_key :account

  one :oauth_access_token, resource: Latest::OauthAccessTokenSerializer
  one :profile, resource: Internal::ProfileSerializer
  one :user, resource: Latest::UserSerializer
end
