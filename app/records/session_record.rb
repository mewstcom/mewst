# typed: strict
# frozen_string_literal: true

class SessionRecord < ApplicationRecord
  self.table_name = "sessions"

  COOKIE_KEY = :mewst_session_token

  has_secure_token

  belongs_to :actor_record, class_name: "ActorRecord", foreign_key: :actor_id

  sig { params(ip_address: T.nilable(String), user_agent: T.nilable(String), signed_in_at: ActiveSupport::TimeWithZone).returns(SessionRecord) }
  def self.start!(ip_address:, user_agent:, signed_in_at: Time.zone.now)
    create!(
      ip_address: ip_address || "",
      user_agent: user_agent || "",
      signed_in_at:
    )
  end
end
