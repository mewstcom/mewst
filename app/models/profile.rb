# typed: strict
# frozen_string_literal: true

class Profile < ApplicationRecord
  extend Enumerize

  include SoftDeletable
  include TimelineOwnable

  ATNAME_FORMAT = /\A[A-Za-z0-9_]+\z/

  enumerize :profileable_type, in: %i[user]

  has_many :actors, dependent: :restrict_with_exception
  has_many :oauth_access_tokens, dependent: :restrict_with_exception, foreign_key: :resource_owner_id, inverse_of: :resource_owner
  has_many :follows, dependent: :restrict_with_exception, foreign_key: :source_profile_id, inverse_of: :source_profile
  has_many :inverse_follows, class_name: "Follow", dependent: :restrict_with_exception, foreign_key: :target_profile_id, inverse_of: :target_profile
  has_many :followees, class_name: "Profile", source: :target_profile, through: :follows
  has_many :followers, class_name: "Profile", source: :source_profile, through: :inverse_follows
  has_many :posts, dependent: :restrict_with_exception
  has_many :stamps, dependent: :restrict_with_exception
  has_one :user, dependent: :restrict_with_exception

  validates :atname, format: {with: ATNAME_FORMAT}, length: {maximum: 20}, presence: true, uniqueness: true

  delegate :delete_post, to: :postability

  sig { returns(String) }
  def name_or_atname
    name.presence || atname
  end

  sig { returns(Profile::HomeTimeline) }
  def home_timeline
    Profile::HomeTimeline.new(profile: self)
  end

  sig { override.returns(String) }
  def timeline_key
    "timeline:profile:#{id}"
  end

  sig do
    params(application: OauthApplication, scopes: T.any(String, Doorkeeper::OAuth::Scopes))
      .returns(T.nilable(OauthAccessToken))
  end
  def active_access_token(application: OauthApplication.mewst_web, scopes: "")
    OauthAccessToken.matching_token_for(application, self, scopes, include_expired: false)
  end

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def following?(target_profile:)
    follows.exists?(target_profile:)
  end

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def me?(target_profile:)
    atname == target_profile.atname
  end

  sig { returns(Profile::Postability) }
  private def postability
    Profile::Postability.new(profile: self)
  end
end
