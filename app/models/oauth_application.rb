# typed: strict
# frozen_string_literal: true

class OauthApplication < Doorkeeper::Application
  MEWST_WEB_UID = "mewst-web"

  def self.mewst_web
    find_by!(uid: MEWST_WEB_UID)
  end
end
