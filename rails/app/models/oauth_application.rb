# typed: strict
# frozen_string_literal: true

class OauthApplication < Doorkeeper::Application
  extend T::Sig

  MEWST_WEB_UID = "mewst-web"

  sig { returns(OauthApplication) }
  def self.mewst_web
    find_by!(uid: MEWST_WEB_UID)
  end
end
