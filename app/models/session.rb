# typed: strict
# frozen_string_literal: true

class Session < ApplicationRecord
  COOKIE_KEY = :mewst_session_token

  has_secure_token

  belongs_to :actor

  sig { params(ip_address: String, user_agent: String, signed_in_at: ActiveSupport::TimeWithZone).returns(Session) }
  def self.start!(ip_address:, user_agent:, signed_in_at: Time.current)
    create!(ip_address:, user_agent:, signed_in_at:)
  end
end
