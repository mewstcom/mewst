# typed: strict
# frozen_string_literal: true

class Actor < ApplicationRecord
  belongs_to :user
  belongs_to :profile
  has_many :oauth_access_tokens, dependent: :restrict_with_exception, foreign_key: :resource_owner_id, inverse_of: :resource_owner

  delegate :time_zone, to: :user
  delegate :atname, :avatar_url, :checkable_suggested_followees, :description, :following?, :follows, :home_timeline,
    :me?, :name, :notifications, :posts, :stamps, :suggested_follows, :suggested_followees,
    to: :profile

  sig { returns(Locale) }
  def locale
    Locale.deserialize(user.not_nil!.locale)
  end

  sig { params(time: ActiveSupport::TimeWithZone).returns(T::Boolean) }
  def update_last_post_time!(time:)
    profile.not_nil!.update!(last_post_at: time)
  end

  sig do
    params(application: OauthApplication, scopes: T.any(String, Doorkeeper::OAuth::Scopes))
      .returns(T.nilable(OauthAccessToken))
  end
  def active_access_token(application: OauthApplication.mewst_web, scopes: "")
    OauthAccessToken.matching_token_for(application, self, scopes, include_expired: false)
  end
end
