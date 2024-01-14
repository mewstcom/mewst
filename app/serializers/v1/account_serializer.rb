# typed: strict
# frozen_string_literal: true

class V1::AccountSerializer < V1::ApplicationSerializer
  root_key :account

  one :oauth_access_token, resource: V1::OauthAccessTokenSerializer
  one :profile, resource: V1::ProfileSerializer, &:profile_resource
  one :user, resource: V1::UserSerializer
end
