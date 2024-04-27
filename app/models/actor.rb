# typed: strict
# frozen_string_literal: true

class Actor < ApplicationRecord
  belongs_to :user
  belongs_to :profile
  has_many :oauth_access_tokens, dependent: :restrict_with_exception, foreign_key: :resource_owner_id, inverse_of: :resource_owner
  has_many :sessions, dependent: :restrict_with_exception

  delegate :email, :time_zone, to: :user
  delegate :atname, :avatar_kind, :avatar_url, :checkable_suggested_followees, :description, :fetch_notifications,
    :following?, :follows, :generate_gravatar_url, :gravatar_email, :gravatar_url, :home_timeline, :image_url, :me?,
    :name, :notifications, :posts, :stamps, :suggested_follows, :suggested_followees,
    to: :profile

  sig { returns(Locale) }
  def locale
    Locale.deserialize(user.not_nil!.locale)
  end

  sig { params(time: ActiveSupport::TimeWithZone).returns(T::Boolean) }
  def update_last_post_time!(time:)
    profile.not_nil!.update!(last_post_at: time)
  end

  sig { params(email: String).returns(T::Boolean) }
  def update_email!(email:)
    user.not_nil!.update!(email:)
  end

  sig do
    params(application: OauthApplication, scopes: T.any(String, Doorkeeper::OAuth::Scopes))
      .returns(T.nilable(OauthAccessToken))
  end
  def active_access_token(application: OauthApplication.mewst_web, scopes: "")
    OauthAccessToken.matching_token_for(application, self, scopes, include_expired: false)
  end
end
