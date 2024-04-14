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

  validates :content, length: {maximum: MAXIMUM_CONTENT_LENGTH}, presence: true

  sig { returns(String) }
  def timeline_score
    published_at.strftime("%s%L")
  end

  sig { params(profile: Profile).returns(T::Boolean) }
  def stamped_by?(profile:)
    stamps.include?(profile:)
  end
end
