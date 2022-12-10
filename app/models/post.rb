# typed: strict
# frozen_string_literal: true

class Post < ApplicationRecord
  CONTENT_MAXIMUM_LENGTH = 500

  belongs_to :profile

  sig { returns(String) }
  def timeline_item_score
    T.must(created_at).to_datetime.strftime("%Q")
  end
end
