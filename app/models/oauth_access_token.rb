# typed: strict
# frozen_string_literal: true

class OauthAccessToken < Doorkeeper::AccessToken
  belongs_to :resource_owner, class_name: "User"

  alias_attribute :user, :resource_owner
end
