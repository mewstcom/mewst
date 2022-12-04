# typed: strict
# frozen_string_literal: true

class Post < ApplicationRecord
  belongs_to :profile

  def inbox_item_score
    created_at.to_datetime.strftime("%Q")
  end
end
