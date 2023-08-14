# typed: strict
# frozen_string_literal: true

class OauthAccessToken < Doorkeeper::AccessToken
  extend T::Sig

  belongs_to :resource_owner, class_name: "Profile"
  belongs_to :user

  alias_attribute :profile, :resource_owner
end
