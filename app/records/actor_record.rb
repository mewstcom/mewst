# typed: strict
# frozen_string_literal: true

class ActorRecord < ApplicationRecord
  self.table_name = "actors"

  belongs_to :user_record, class_name: "UserRecord", foreign_key: :user_id
  belongs_to :profile_record, class_name: "ProfileRecord", foreign_key: :profile_id
  has_many :oauth_access_tokens, dependent: :restrict_with_exception, foreign_key: :resource_owner_id, inverse_of: :resource_owner
  has_many :session_records, class_name: "SessionRecord", dependent: :restrict_with_exception

  delegate :email, :time_zone, to: :user_record
  delegate :atname, :avatar_kind, :avatar_url, :checkable_suggested_followees, :description, :fetch_notifications,
    :following?, :generate_gravatar_url, :gravatar_email, :gravatar_url,
    :image_url, :me?, :name,
    to: :profile_record

  sig { returns(Locale) }
  def locale
    Locale.deserialize(user_record.not_nil!.locale)
  end

  sig { params(time: ActiveSupport::TimeWithZone).returns(T::Boolean) }
  def update_last_post_time!(time:)
    profile_record.not_nil!.update!(last_post_at: time)
  end

  sig { params(email: String).returns(T::Boolean) }
  def update_email!(email:)
    user_record.not_nil!.update!(email:)
  end

  sig do
    params(application: OauthApplication, scopes: T.any(String, Doorkeeper::OAuth::Scopes))
      .returns(T.nilable(OauthAccessToken))
  end
  def active_access_token(application: OauthApplication.mewst_web, scopes: "")
    OauthAccessToken.matching_token_for(application, self, scopes, include_expired: false)
  end
end
