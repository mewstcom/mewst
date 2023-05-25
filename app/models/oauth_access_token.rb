# typed: strict
# frozen_string_literal: true

class OauthAccessToken < Doorkeeper::AccessToken
  belongs_to :account
end
