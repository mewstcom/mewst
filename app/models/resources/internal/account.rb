# typed: strict
# frozen_string_literal: true

class Resources::Internal::Account < Resources::Internal::Base
  root_key :account

  one :oauth_access_token, resource: Resources::Internal::OauthAccessToken
  one :profile, resource: Resources::Internal::Profile
  one :user, resource: Resources::Internal::User
end
