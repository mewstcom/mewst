# typed: strict
# frozen_string_literal: true

class HomeTimelinePostRecord < ApplicationRecord
  self.table_name = "home_timeline_posts"

  belongs_to :post
  belongs_to :profile

  scope :visible, -> { where(published_at: 1.month.ago..) }
  scope :invisible, -> { where(published_at: ...1.month.ago) }
end