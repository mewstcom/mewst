# typed: strict
# frozen_string_literal: true

class HomeTimelinePostRecord < ApplicationRecord
  self.table_name = "home_timeline_posts"

  belongs_to :post_record, class_name: "PostRecord", foreign_key: :post_id
  belongs_to :profile_record, class_name: "ProfileRecord", foreign_key: :profile_id

  scope :visible, -> { where(published_at: 1.month.ago..) }
  scope :invisible, -> { where(published_at: ...1.month.ago) }
end
