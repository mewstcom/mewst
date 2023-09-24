# typed: strict
# frozen_string_literal: true

class OauthAccessToken < Doorkeeper::AccessToken
  extend T::Sig

  belongs_to :resource_owner, class_name: "Actor"
end
