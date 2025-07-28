# typed: strict
# frozen_string_literal: true

class PostRecord < ApplicationRecord
  self.table_name = "posts"

  extend Enumerize

  include Discard::Model

  # ref: https://techcrunchjapan.com/2009/07/06/20090704short-is-sweet-postcards-begat-sms-begat-twitter/
  # Note: なぜ140文字ではないのか？
  #   1. SMSに送ることは想定していないので、20文字を削る必要がないため
  #   2. URLが含まれると140文字だと心もとないため
  MAXIMUM_CONTENT_LENGTH = 160

  belongs_to :profile_record, class_name: "ProfileRecord", foreign_key: :profile_id
  belongs_to :oauth_application
  has_many :stamp_records, class_name: "StampRecord", dependent: :restrict_with_exception, foreign_key: :post_id
  has_many :home_timeline_post_records, class_name: "HomeTimelinePostRecord", dependent: :restrict_with_exception, foreign_key: :post_id
  has_one :post_link_record, class_name: "PostLinkRecord", dependent: :restrict_with_exception, foreign_key: :post_id
  has_one :link_record, class_name: "LinkRecord", through: :post_link_record

  scope :kept, -> { undiscarded.joins(:profile_record).merge(ProfileRecord.kept) }
  scope :order_by_recent, -> { order(published_at: :desc, id: :desc) }
  scope :order_by_oldest, -> { order(published_at: :asc, id: :asc) }

  validates :content, length: {maximum: MAXIMUM_CONTENT_LENGTH}, presence: true

  sig { params(profile: ProfileRecord).returns(T::Boolean) }
  def stamped_by?(profile:)
    stamp_records.exists?(profile_id: profile.id)
  end

  sig { returns(T.nilable(PostRecord)) }
  def prev_post
    profile_record.not_nil!.posts.kept.where(id: ...id).order_by_recent.first
  end

  sig { returns(T.nilable(PostRecord)) }
  def next_post
    profile_record.not_nil!.posts.kept.where(PostRecord.arel_table[:id].gt(id)).order_by_oldest.first
  end
end
