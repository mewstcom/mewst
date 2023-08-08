# typed: strict
# frozen_string_literal: true

class Internal::AccountSerializer < Internal::ApplicationSerializer
  has_one :oauth_access_token, serializer: Internal::OauthAccessTokenSerializer
  has_one :profile, serializer: Internal::ProfileSerializer
  has_one :user, serializer: Internal::UserSerializer
end
