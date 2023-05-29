# typed: strict
# frozen_string_literal: true

class OauthAccessToken < Doorkeeper::AccessToken
  belongs_to :account
  belongs_to :profile, foreign_key: :resource_owner_id
end
