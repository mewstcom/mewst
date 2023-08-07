# typed: strict
# frozen_string_literal: true

class Internal::Resources::Account < Internal::Resources::Base
  root_key :account

  one :oauth_access_token, resource: Internal::Resources::OauthAccessToken
  one :profile, resource: Internal::Resources::Profile
  one :user, resource: Internal::Resources::User
end
