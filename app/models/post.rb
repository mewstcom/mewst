# typed: strict
# frozen_string_literal: true

class Post < ApplicationRecord
  extend Enumerize

  include Discard::Model

  # ref: https://techcrunchjapan.com/2009/07/06/20090704short-is-sweet-postcards-begat-sms-begat-twitter/
  # Note: なぜ140文字ではないのか？
  #   1. SMSに送ることは想定していないので、20文字を削る必要がないため
  #   2. URLが含まれると140文字だと心もとないため
  MAXIMUM_CONTENT_LENGTH = 160

  belongs_to :profile
  belongs_to :oauth_application
  has_many :stamps, dependent: :restrict_with_exception
  has_many :home_timeline_posts, dependent: :restrict_with_exception
  has_one :post_link, dependent: :restrict_with_exception
  has_one :link, through: :post_link

  scope :kept, -> { undiscarded.joins(:profile).merge(Profile.kept) }
  scope :order_by_recent, -> { order(published_at: :desc, id: :desc) }
  scope :order_by_oldest, -> { order(published_at: :asc, id: :asc) }

  validates :content, length: {maximum: MAXIMUM_CONTENT_LENGTH}, presence: true

  sig { params(profile: Profile).returns(T::Boolean) }
  def stamped_by?(profile:)
    stamps.include?(profile:)
  end

  sig { returns(T.nilable(Post)) }
  def prev_post
    profile.posts.kept.where(id: ...id).order_by_recent.first
  end

  sig { returns(T.nilable(Post)) }
  def next_post
    profile.posts.kept.where(Post.arel_table[:id].gt(id)).order_by_oldest.first
  end
end
