# typed: strict
# frozen_string_literal: true

class Actor < ApplicationRecord
  belongs_to :user
  belongs_to :profile
  has_many :oauth_access_tokens, dependent: :restrict_with_exception, foreign_key: :resource_owner_id, inverse_of: :resource_owner

  delegate :atname, :following?, :follows, :home_timeline, :me?, :notifications, :posts, :stamps,
    :suggested_follows, :suggested_followees,
    to: :profile

  sig do
    params(application: OauthApplication, scopes: T.any(String, Doorkeeper::OAuth::Scopes))
      .returns(T.nilable(OauthAccessToken))
  end
  def active_access_token(application: OauthApplication.mewst_web, scopes: "")
    OauthAccessToken.matching_token_for(application, self, scopes, include_expired: false)
  end
end
