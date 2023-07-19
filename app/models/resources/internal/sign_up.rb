# typed: strict
# frozen_string_literal: true

class Resources::Internal::SignUp < Resources::Internal::Base
  root_key :sign_up

  one :oauth_access_token, resource: Resources::Internal::OauthAccessToken
  one :profile, resource: Resources::Internal::Profile
  one :user, resource: Resources::Internal::User
end
